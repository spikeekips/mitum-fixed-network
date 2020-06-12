package isaac

import (
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

/*
StateConsensusHandler joins network consensus.

What does consensus state means?

- Block states are synced with the network.
- Node can participate every vote stages.

Consensus state is started by new INIT Voteproof and waits next Proposal.
*/
type StateConsensusHandler struct {
	proposalLock sync.Mutex
	*BaseStateHandler
	suffrage      base.Suffrage
	proposalMaker *ProposalMaker
}

func NewStateConsensusHandler(
	localstate *Localstate,
	proposalProcessor ProposalProcessor,
	suffrage base.Suffrage,
	proposalMaker *ProposalMaker,
) (*StateConsensusHandler, error) {
	cs := &StateConsensusHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, proposalProcessor, base.StateConsensus),
		suffrage:         suffrage,
		proposalMaker:    proposalMaker,
	}
	cs.BaseStateHandler.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "consensus-state-consensus-handler")
	})
	cs.timers = localtime.NewTimers(
		[]string{
			TimerIDBroadcastingINITBallot,
			TimerIDBroadcastingACCEPTBallot,
			TimerIDBroadcastingProposal,
			TimerIDTimedoutMoveNextRound,
		},
		false,
	)

	return cs, nil
}

func (cs *StateConsensusHandler) SetLogger(l logging.Logger) logging.Logger {
	_ = cs.Logging.SetLogger(l)
	_ = cs.timers.SetLogger(l)

	return cs.Log()
}

func (cs *StateConsensusHandler) Activate(ctx StateChangeContext) error {
	if _, found, err := cs.localstate.Storage().LastManifest(); !found {
		return storage.NotFoundError.Errorf("last manifest is empty")
	} else if err != nil {
		return xerrors.Errorf("failed to get last manifest: %w", err)
	}

	if ctx.Voteproof() == nil {
		return xerrors.Errorf("consensus handler got empty Voteproof")
	} else if ctx.Voteproof().Stage() != base.StageINIT {
		return xerrors.Errorf("consensus handler starts with INIT Voteproof: %s", ctx.Voteproof().Stage())
	} else if err := ctx.Voteproof().IsValid(cs.localstate.Policy().NetworkID()); err != nil {
		return xerrors.Errorf("consensus handler got invalid Voteproof: %w", err)
	}

	l := loggerWithStateChangeContext(ctx, cs.Log())

	go func() {
		if err := cs.handleINITVoteproof(ctx.Voteproof()); err != nil {
			l.Error().Err(err).Msg("activated, but handleINITVoteproof failed with voteproof")
		}
	}()

	l.Debug().Msg("activated")

	return nil
}

func (cs *StateConsensusHandler) Deactivate(ctx StateChangeContext) error {
	l := loggerWithStateChangeContext(ctx, cs.Log())

	if err := cs.timers.Stop(); err != nil {
		return err
	}

	l.Debug().Msg("deactivated")

	return nil
}

func (cs *StateConsensusHandler) waitProposal(voteproof base.Voteproof) error {
	cs.proposalLock.Lock()
	defer cs.proposalLock.Unlock()

	cs.Log().Debug().Msg("waiting proposal")

	if proposed, err := cs.proposal(voteproof); err != nil {
		return err
	} else if proposed {
		return nil
	}

	if err := cs.checkReceivedProposal(voteproof.Height(), voteproof.Round()); err != nil {
		return err
	}

	if timer, err := cs.TimerTimedoutMoveNextRound(voteproof); err != nil {
		return err
	} else if err := cs.timers.SetTimer(TimerIDTimedoutMoveNextRound, timer); err != nil {
		return err
	}

	return cs.timers.StartTimers([]string{
		TimerIDTimedoutMoveNextRound,
		TimerIDBroadcastingINITBallot, // keep broadcasting when waiting
		TimerIDBroadcastingACCEPTBallot,
	}, true)
}

func (cs *StateConsensusHandler) NewSeal(sl seal.Seal) error {
	switch t := sl.(type) {
	case ballot.Proposal:
		go func(proposal ballot.Proposal) {
			if err := cs.handleProposal(proposal); err != nil {
				cs.Log().Error().Err(err).
					Hinted("proposal", proposal.Hash()).
					Msg("failed to handle proposal")
			}
		}(t)

		return nil
	default:
		return nil
	}
}

func (cs *StateConsensusHandler) NewVoteproof(voteproof base.Voteproof) error {
	l := loggerWithVoteproof(voteproof, cs.Log())

	l.Debug().Msg("got Voteproof")

	l.Debug().Msg("got Voteproof; for restarting next round timer")
	if timer, err := cs.TimerTimedoutMoveNextRound(voteproof); err != nil {
		return err
	} else if err := cs.timers.SetTimer(TimerIDTimedoutMoveNextRound, timer); err != nil {
		return err
	} else if err := cs.timers.StartTimers([]string{
		TimerIDTimedoutMoveNextRound,
		TimerIDBroadcastingINITBallot,
		TimerIDBroadcastingACCEPTBallot,
	}, true); err != nil {
		return err
	}

	// NOTE if drew, goes to next round.
	if voteproof.Result() == base.VoteResultDraw {
		return cs.startNextRound(voteproof)
	}

	switch voteproof.Stage() {
	case base.StageACCEPT:
		if err := util.Retry(3, time.Millisecond*200, func() error {
			if err := cs.StoreNewBlock(voteproof); err != nil {
				l.Error().Err(err).Msg("something wrong to store accept voteproof; will retry")

				return err
			}

			return nil
		}); err != nil {
			l.Error().Err(err).Msg("failed to store accept voteproof; moves to sync")

			return cs.ChangeState(base.StateSyncing, voteproof, nil)
		}

		return cs.keepBroadcastingINITBallotForNextBlock(voteproof)
	case base.StageINIT:
		return cs.handleINITVoteproof(voteproof)
	default:
		err := xerrors.Errorf("invalid Voteproof received")

		l.Error().Err(err).Msg("invalid voteproof found")

		return err
	}
}

func (cs *StateConsensusHandler) handleINITVoteproof(voteproof base.Voteproof) error {
	l := loggerWithLocalstate(cs.localstate, loggerWithVoteproof(voteproof, cs.Log()))

	l.Debug().Msg("expected Voteproof received; will wait Proposal")

	return cs.waitProposal(voteproof)
}

func (cs *StateConsensusHandler) keepBroadcastingINITBallotForNextBlock(voteproof base.Voteproof) error {
	if timer, err := cs.TimerBroadcastingINITBallot(
		func() time.Duration { return cs.localstate.Policy().IntervalBroadcastingINITBallot() },
		func() base.Round { return base.Round(0) },
		voteproof,
	); err != nil {
		return err
	} else if err := cs.timers.SetTimer(TimerIDBroadcastingINITBallot, timer); err != nil {
		return err
	}

	return cs.timers.StartTimers([]string{
		TimerIDBroadcastingINITBallot,
		TimerIDBroadcastingACCEPTBallot,
	}, true)
}

func (cs *StateConsensusHandler) handleProposal(proposal ballot.Proposal) error {
	cs.proposalLock.Lock()
	defer cs.proposalLock.Unlock()

	l := loggerWithBallot(proposal, cs.Log())

	l.Debug().Msg("got proposal")

	blk, err := cs.proposalProcessor.ProcessINIT(proposal.Hash(), cs.LastINITVoteproof())
	if err != nil {
		return err
	}

	acting := cs.suffrage.Acting(proposal.Height(), proposal.Round())
	isActing := acting.Exists(cs.localstate.Node().Address())

	l.Debug().
		Hinted("acting_suffrage", acting).
		Bool("is_acting", isActing).
		Msgf("node is in acting suffrage? %v", isActing)

	if isActing {
		if err := cs.readyToSIGNBallot(blk); err != nil {
			return err
		}
	}

	return cs.readyToACCEPTBallot(blk)
}

func (cs *StateConsensusHandler) readyToSIGNBallot(newBlock block.Block) error {
	// NOTE not like broadcasting ACCEPT Ballot, SIGN Ballot will be broadcasted
	// withtout waiting.

	sb := NewSIGNBallotV0(cs.localstate.Node().Address(), newBlock)
	if err := SignSeal(&sb, cs.localstate); err != nil {
		return err
	} else {
		cs.BroadcastSeal(sb)

		loggerWithBallot(sb, cs.Log()).Debug().Msg("SIGNBallot was broadcasted")
	}

	return nil
}

func (cs *StateConsensusHandler) readyToACCEPTBallot(newBlock block.Block) error {
	// NOTE if not in acting suffrage, broadcast ACCEPT Ballot after interval.
	if timer, err := cs.TimerBroadcastingACCEPTBallot(newBlock); err != nil {
		return err
	} else if err := cs.timers.SetTimer(TimerIDBroadcastingACCEPTBallot, timer); err != nil {
		return err
	}

	return cs.timers.StartTimers([]string{
		TimerIDTimedoutMoveNextRound,
		TimerIDBroadcastingACCEPTBallot,
	}, true)
}

func (cs *StateConsensusHandler) proposal(voteproof base.Voteproof) (bool, error) {
	l := loggerWithVoteproof(voteproof, cs.Log())

	l.Debug().Msg("prepare to broadcast Proposal")
	isProposer := cs.suffrage.IsProposer(voteproof.Height(), voteproof.Round(), cs.localstate.Node().Address())
	l.Debug().
		Hinted("acting_suffrage", cs.suffrage.Acting(voteproof.Height(), voteproof.Round())).
		Bool("is_acting", cs.suffrage.IsActing(voteproof.Height(), voteproof.Round(), cs.localstate.Node().Address())).
		Bool("is_proposer", isProposer).
		Msgf("node is proposer? %v", isProposer)

	if !isProposer {
		return false, nil
	}

	proposal, err := cs.proposalMaker.Proposal(voteproof.Round())
	if err != nil {
		return false, xerrors.Errorf("failed to make proposal: %w", err)
	}

	l.Debug().Interface("proposal", proposal).Msg("trying to broadcast Proposal")

	if timer, err := cs.TimerBroadcastingProposal(proposal); err != nil {
		return false, err
	} else if err := cs.timers.SetTimer(TimerIDBroadcastingProposal, timer); err != nil {
		return false, err
	} else if err := cs.timers.StartTimers(
		[]string{
			TimerIDTimedoutMoveNextRound,
			TimerIDBroadcastingProposal,
			TimerIDBroadcastingINITBallot,
		}, true,
	); err != nil {
		return false, err
	}

	return true, nil
}

func (cs *StateConsensusHandler) startNextRound(voteproof base.Voteproof) error {
	cs.Log().Debug().Msg("trying to start next round")

	var round base.Round
	if voteproof.Stage() == base.StageACCEPT {
		round = 0
	} else {
		round = voteproof.Round() + 1
	}

	var called int64
	if timer, err := cs.TimerBroadcastingINITBallot(
		func() time.Duration {
			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast INIT Ballot.
			if atomic.LoadInt64(&called) < 1 {
				atomic.AddInt64(&called, 1)
				return time.Nanosecond
			}

			return cs.localstate.Policy().IntervalBroadcastingINITBallot()
		},
		func() base.Round { return round },
		voteproof,
	); err != nil {
		return err
	} else if err := cs.timers.SetTimer(TimerIDBroadcastingINITBallot, timer); err != nil {
		return err
	}

	return cs.timers.StartTimers([]string{
		TimerIDBroadcastingINITBallot,
		TimerIDBroadcastingACCEPTBallot,
	}, true)
}

func (cs *StateConsensusHandler) checkReceivedProposal(height base.Height, round base.Round) error {
	cs.Log().Debug().Msg("trying to check already received Proposal")

	// if Proposal already received, find and processing it.
	proposal, found, err := cs.localstate.Storage().Proposal(height, round)
	if !found {
		return nil
	} else if err != nil {
		cs.Log().Error().Err(err).Msg("expected Proposal not found, but keep trying")

		return err
	}

	go func() {
		if err := cs.handleProposal(proposal); err != nil {
			cs.Log().Error().Err(err).Msg("processing already received proposal, but")
		}
	}()

	return nil
}
