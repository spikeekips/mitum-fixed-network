package isaac

import (
	"context"
	"sync"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/prprocessor"
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
	local *Local,
	pps *prprocessor.Processors,
	suffrage base.Suffrage,
	proposalMaker *ProposalMaker,
) (*StateConsensusHandler, error) {
	cs := &StateConsensusHandler{
		BaseStateHandler: NewBaseStateHandler(local, pps, base.StateConsensus),
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

func (cs *StateConsensusHandler) Activate(ctx *StateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	if ctx == nil {
		return xerrors.Errorf("empty StateChangeContext")
	}

	if _, found, err := cs.local.Storage().LastManifest(); !found {
		return storage.NotFoundError.Errorf("last manifest is empty")
	} else if err != nil {
		return xerrors.Errorf("failed to get last manifest: %w", err)
	}

	if ctx.Voteproof() == nil {
		return xerrors.Errorf("consensus handler got empty Voteproof")
	} else if ctx.Voteproof().Stage() != base.StageINIT {
		return xerrors.Errorf("consensus handler starts with INIT Voteproof: %s", ctx.Voteproof().Stage())
	} else if err := ctx.Voteproof().IsValid(cs.local.Policy().NetworkID()); err != nil {
		return xerrors.Errorf("consensus handler got invalid Voteproof: %w", err)
	}

	cs.activate()

	go func() {
		if err := cs.handleINITVoteproof(ctx.Voteproof()); err != nil {
			cs.Log().Error().Err(err).Msg("activated, but handleINITVoteproof failed with voteproof")
		}
	}()

	cs.Log().Debug().Msg("activated")

	return nil
}

func (cs *StateConsensusHandler) Deactivate(_ *StateChangeContext) error {
	cs.deactivate()

	if err := cs.timers.Stop(); err != nil {
		return err
	}

	cs.Log().Debug().Msg("deactivated")

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
				if xerrors.Is(err, util.IgnoreError) {
					return
				}

				cs.Log().Error().Err(err).
					Hinted("proposal_hash", proposal.Hash()).
					Msg("failed to handle proposal")
			}
		}(t)

		return nil
	default:
		return nil
	}
}

func (cs *StateConsensusHandler) NewVoteproof(voteproof base.Voteproof) error {
	l := loggerWithVoteproofID(voteproof, cs.Log())

	if voteproof.Result() == base.VoteResultDraw { // NOTE if drew, goes to next round.
		return cs.startNextRound(voteproof)
	}

	l.Debug().Msg("got Voteproof")
	var nVoteproof base.Voteproof
	switch voteproof.Stage() {
	case base.StageINIT:
		nVoteproof = voteproof
	case base.StageACCEPT:
		nVoteproof = cs.LastINITVoteproof()
	}

	if timer, err := cs.TimerTimedoutMoveNextRound(nVoteproof); err != nil {
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

	return cs.newVoteproof(voteproof)
}

func (cs *StateConsensusHandler) newVoteproof(voteproof base.Voteproof) error {
	l := loggerWithVoteproofID(voteproof, cs.Log())

	switch voteproof.Stage() {
	case base.StageACCEPT:
		return cs.handleACCEPTVoteproof(voteproof)
	case base.StageINIT:
		return cs.handleINITVoteproof(voteproof)
	default:
		err := xerrors.Errorf("invalid Voteproof received")

		l.Error().Err(err).Msg("invalid voteproof found")

		return err
	}
}

func (cs *StateConsensusHandler) handleINITVoteproof(voteproof base.Voteproof) error {
	l := loggerWithLocal(cs.local, loggerWithVoteproofID(voteproof, cs.Log()))

	l.Debug().Msg("expected Voteproof received; will wait Proposal")

	return cs.waitProposal(voteproof)
}

func (cs *StateConsensusHandler) handleACCEPTVoteproof(voteproof base.Voteproof) error {
	l := loggerWithLocal(cs.local, loggerWithVoteproofID(voteproof, cs.Log()))

	if err := cs.StoreNewBlock(voteproof); err != nil {
		var ctx *StateToBeChangeError
		switch {
		case xerrors.Is(err, util.IgnoreError):
			l.Error().Err(err).Msg("accept voteproof will be ignored")

			return nil
		case xerrors.Is(err, storage.TimeoutError):
			l.Error().Err(err).Msg("failed to store accept voteproof with timeout error; moves to next round")

			return err
		case xerrors.As(err, &ctx):
			l.Error().Err(err).Msg("state will be moved with accept voteproof")

			return cs.ChangeState(ctx.ToState, ctx.Voteproof, ctx.Ballot)
		default:
			if len(cs.suffrage.Nodes()) < 2 {
				l.Error().Err(err).Msg("failed to store accept voteproof; standalone node will wait")

				return err
			}

			l.Error().Err(err).Msg("failed to store accept voteproof; moves to sync")

			return cs.ChangeState(base.StateSyncing, voteproof, nil)
		}
	}

	return cs.keepBroadcastingINITBallotForNextBlock(voteproof)
}

func (cs *StateConsensusHandler) keepBroadcastingINITBallotForNextBlock(voteproof base.Voteproof) error {
	if timer, err := cs.TimerBroadcastingINITBallot(
		func(int) time.Duration { return cs.local.Policy().IntervalBroadcastingINITBallot() },
		base.Round(0),
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

	if err := cs.timers.ResetTimer(TimerIDTimedoutMoveNextRound); err != nil {
		l.Debug().Err(err).Str("timer", TimerIDTimedoutMoveNextRound).Msg("tried to reset timer, but failed; ignored")
	}

	voteproof := cs.LastINITVoteproof()

	timeout := cs.local.Policy().TimeoutProcessProposal()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cs.Log().Debug().Dur("timeout", timeout).Msg("trying to prepare block")

	var newBlock block.Block
	if result := <-cs.pps.NewProposal(ctx, proposal, voteproof); result.Err != nil {
		return result.Err
	} else {
		newBlock = result.Block
	}

	acting := cs.suffrage.Acting(proposal.Height(), proposal.Round())
	isActing := acting.Exists(cs.local.Node().Address())

	l.Debug().
		Hinted("acting_suffrage", acting).
		Bool("is_acting", isActing).
		Msgf("node is in acting suffrage? %v", isActing)

	if isActing {
		if err := cs.readyToSIGNBallot(newBlock); err != nil {
			return err
		}
	}

	return cs.readyToACCEPTBallot(newBlock, voteproof)
}

func (cs *StateConsensusHandler) readyToSIGNBallot(newBlock block.Block) error {
	// NOTE not like broadcasting ACCEPT Ballot, SIGN Ballot will be broadcasted
	// withtout waiting.

	sb := NewSIGNBallotV0(cs.local.Node().Address(), newBlock)
	if err := SignSeal(&sb, cs.local); err != nil {
		return err
	} else {
		cs.BroadcastSeal(sb)

		loggerWithBallot(sb, cs.Log()).Debug().Msg("SIGNBallot was broadcasted")
	}

	return nil
}

func (cs *StateConsensusHandler) readyToACCEPTBallot(newBlock block.Block, voteproof base.Voteproof) error {
	// NOTE if not in acting suffrage, broadcast ACCEPT Ballot after interval.
	if timer, err := cs.TimerBroadcastingACCEPTBallot(newBlock, voteproof); err != nil {
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
	l := loggerWithVoteproofID(voteproof, cs.Log())

	l.Debug().Msg("prepare to broadcast Proposal")
	isProposer := cs.suffrage.IsProposer(voteproof.Height(), voteproof.Round(), cs.local.Node().Address())
	l.Debug().
		Hinted("acting_suffrage", cs.suffrage.Acting(voteproof.Height(), voteproof.Round())).
		Bool("is_acting", cs.suffrage.IsActing(voteproof.Height(), voteproof.Round(), cs.local.Node().Address())).
		Bool("is_proposer", isProposer).
		Msgf("node is proposer? %v", isProposer)

	if !isProposer {
		return false, nil
	}

	proposal, err := cs.proposalMaker.Proposal(voteproof.Height(), voteproof.Round())
	if err != nil {
		return false, xerrors.Errorf("failed to make proposal: %w", err)
	}

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

	if timer, err := cs.TimerBroadcastingINITBallot(
		func(i int) time.Duration {
			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast INIT Ballot.
			if i < 1 {
				return time.Nanosecond
			}

			return cs.local.Policy().IntervalBroadcastingINITBallot()
		},
		round,
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
	proposal, found, err := cs.local.Storage().Proposal(height, round)
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

func (cs *StateConsensusHandler) SetLastINITVoteproof(voteproof base.Voteproof) error {
	if err := cs.BaseStateHandler.SetLastINITVoteproof(voteproof); err != nil {
		return err
	}

	return nil
}
