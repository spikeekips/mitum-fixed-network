package isaac

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/storage"
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
	suffrage          Suffrage
	proposalMaker     *ProposalMaker
	processedProposal Proposal
}

func NewStateConsensusHandler(
	localstate *Localstate,
	proposalProcessor ProposalProcessor,
	suffrage Suffrage,
	proposalMaker *ProposalMaker,
) (*StateConsensusHandler, error) {
	if lastBlock := localstate.LastBlock(); lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	cs := &StateConsensusHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, proposalProcessor, StateConsensus),
		suffrage:         suffrage,
		proposalMaker:    proposalMaker,
	}
	cs.BaseStateHandler.Logger = logging.NewLogger(func(c zerolog.Context) zerolog.Context {
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

func (cs *StateConsensusHandler) SetLogger(l zerolog.Logger) *logging.Logger {
	_ = cs.Logger.SetLogger(l)
	_ = cs.timers.SetLogger(l)

	return cs.Logger
}

func (cs *StateConsensusHandler) Activate(ctx StateChangeContext) error {
	if ctx.Voteproof() == nil {
		return xerrors.Errorf("consensus handler got empty Voteproof")
	} else if ctx.Voteproof().Stage() != StageINIT {
		return xerrors.Errorf("consensus handler starts with INIT Voteproof: %s", ctx.Voteproof().Stage())
	} else if err := ctx.Voteproof().IsValid(nil); err != nil {
		return xerrors.Errorf("consensus handler got invalid Voteproof: %w", err)
	}

	_ = cs.localstate.SetLastINITVoteproof(ctx.Voteproof())

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

func (cs *StateConsensusHandler) waitProposal(vp Voteproof) error { // nolint
	cs.proposalLock.Lock()
	defer cs.proposalLock.Unlock()

	cs.Log().Debug().Msg("waiting proposal")

	if cs.processedProposal != nil {
		if vp.Height() == cs.processedProposal.Height() && vp.Round() == cs.processedProposal.Round() {
			cs.Log().Debug().Msg("proposal is already processed")
			return nil
		}
	}

	if proposed, err := cs.proposal(vp); err != nil {
		return err
	} else if proposed {
		return nil
	}

	if err := cs.checkReceivedProposal(vp.Height(), vp.Round()); err != nil {
		return err
	}

	if timer, err := cs.TimerTimedoutMoveNextRound(vp.Round() + 1); err != nil {
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
	case Proposal:
		go func(proposal Proposal) {
			if err := cs.handleProposal(proposal); err != nil {
				cs.Log().Error().Err(err).
					Str("proposal", proposal.Hash().String()).
					Msg("failed to handle proposal")
			}
		}(t)

		return nil
	default:
		return nil
	}
}

func (cs *StateConsensusHandler) NewVoteproof(vp Voteproof) error {
	if err := cs.timers.StopTimers([]string{TimerIDTimedoutMoveNextRound}); err != nil {
		return err
	}

	l := loggerWithVoteproof(vp, cs.Log())

	l.Debug().Msg("Voteproof received")

	// NOTE if drew, goes to next round.
	if vp.Result() == VoteproofDraw {
		return cs.startNextRound(vp)
	}

	switch vp.Stage() {
	case StageACCEPT:
		if err := cs.StoreNewBlockByVoteproof(vp); err != nil {
			l.Error().Err(err).Msg("failed to store accept voteproof")
		}

		return cs.keepBroadcastingINITBallotForNextBlock()
	case StageINIT:
		return cs.handleINITVoteproof(vp)
	default:
		err := xerrors.Errorf("invalid Voteproof received")

		l.Error().Err(err).Msg("invalid voteproof found")

		return err
	}
}

func (cs *StateConsensusHandler) handleINITVoteproof(vp Voteproof) error {
	l := loggerWithLocalstate(cs.localstate, loggerWithVoteproof(vp, cs.Log()))

	l.Debug().Msg("expected Voteproof received; will wait Proposal")

	return cs.waitProposal(vp)
}

func (cs *StateConsensusHandler) keepBroadcastingINITBallotForNextBlock() error {
	if timer, err := cs.TimerBroadcastingINITBallot(
		func() time.Duration { return cs.localstate.Policy().IntervalBroadcastingINITBallot() },
		func() Round { return Round(0) },
		nil,
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

func (cs *StateConsensusHandler) handleProposal(proposal Proposal) error {
	cs.proposalLock.Lock()
	defer cs.proposalLock.Unlock()

	l := loggerWithBallot(proposal, cs.Log())
	// l := cs.Log()

	l.Debug().Msg("got proposal")

	if cs.processedProposal != nil {
		if proposal.Height() == cs.processedProposal.Height() && proposal.Round() == cs.processedProposal.Round() {
			l.Debug().Msg("proposal is already processed")
			return nil
		}
	}

	// TODO if processing takes too long?
	block, err := cs.proposalProcessor.ProcessINIT(
		proposal.Hash(),
		cs.localstate.LastINITVoteproof(),
		nil,
	)
	if err != nil {
		return err
	}

	if err := cs.timers.StopTimers([]string{TimerIDTimedoutMoveNextRound}); err != nil {
		return err
	} else {
		cs.processedProposal = proposal
	}

	acting := cs.suffrage.Acting(proposal.Height(), proposal.Round())
	isActing := acting.Exists(cs.localstate.Node().Address())

	l.Debug().
		Object("acting_suffrag", acting).
		Bool("is_acting", isActing).
		Msgf("node is in acting suffrage? %v", isActing)

	if isActing {
		if err := cs.readyToSIGNBallot(proposal, block); err != nil {
			return err
		}
	}

	return cs.readyToACCEPTBallot(block)
}

func (cs *StateConsensusHandler) readyToSIGNBallot(proposal Proposal, newBlock Block) error {
	// NOTE not like broadcasting ACCEPT Ballot, SIGN Ballot will be broadcasted
	// withtout waiting.

	sb, err := NewSIGNBallotV0FromLocalstate(cs.localstate, proposal.Round(), newBlock, nil)
	if err != nil {
		cs.Log().Error().Err(err).Msg("failed to create SIGNBallot")
		return err
	}

	cs.BroadcastSeal(sb)

	loggerWithBallot(sb, cs.Log()).Debug().Msg("SIGNBallot was broadcasted")

	return nil
}

func (cs *StateConsensusHandler) readyToACCEPTBallot(newBlock Block) error {
	// NOTE if not in acting suffrage, broadcast ACCEPT Ballot after interval.
	if timer, err := cs.TimerBroadcastingACCEPTBallot(newBlock, nil); err != nil {
		return err
	} else if err := cs.timers.SetTimer(TimerIDBroadcastingACCEPTBallot, timer); err != nil {
		return err
	}

	return cs.timers.StartTimers([]string{TimerIDBroadcastingACCEPTBallot}, true)
}

func (cs *StateConsensusHandler) proposal(vp Voteproof) (bool, error) {
	l := loggerWithVoteproof(vp, cs.Log())

	l.Debug().Msg("prepare to broadcast Proposal")
	isProposer := cs.suffrage.IsProposer(vp.Height(), vp.Round(), cs.localstate.Node().Address())
	l.Debug().
		Object("acting_suffrag", cs.suffrage.Acting(vp.Height(), vp.Round())).
		Bool("is_acting", cs.suffrage.IsActing(vp.Height(), vp.Round(), cs.localstate.Node().Address())).
		Bool("is_proposer", isProposer).
		Msgf("node is proposer? %v", isProposer)

	if !isProposer {
		return false, nil
	}

	proposal, err := cs.proposalMaker.Proposal(vp.Round(), nil)
	if err != nil {
		return false, err
	}

	l.Debug().Interface("proposal", proposal).Msg("trying to broadcast Proposal")

	if timer, err := cs.TimerBroadcastingProposal(proposal); err != nil {
		return false, err
	} else if err := cs.timers.SetTimer(TimerIDBroadcastingProposal, timer); err != nil {
		return false, err
	} else if err := cs.timers.StartTimers(
		[]string{TimerIDBroadcastingProposal, TimerIDBroadcastingINITBallot}, true,
	); err != nil {
		return false, err
	}

	return true, nil
}

func (cs *StateConsensusHandler) startNextRound(vp Voteproof) error {
	cs.Log().Debug().Msg("trying to start next round")

	var round Round
	if vp.Stage() == StageACCEPT {
		round = 0
	} else {
		round = vp.Round() + 1
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
		func() Round { return round },
		nil,
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

func (cs *StateConsensusHandler) checkReceivedProposal(height Height, round Round) error {
	cs.Log().Debug().Msg("trying to check already received Proposal")

	// if Proposal already received, find and processing it.
	proposal, err := cs.localstate.Storage().Proposal(height, round)
	if err != nil {
		if xerrors.Is(err, storage.NotFoundError) {
			return nil
		}

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
