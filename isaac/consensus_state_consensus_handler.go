package isaac

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
)

/*
ConsensusStateConsensusHandler joins network consensus.

What does consensus state means?

- Block states are synced with the network.
- Node can participate every vote stages.

Consensus state is started by new INIT VoteProof and waits next Proposal.
*/
type ConsensusStateConsensusHandler struct {
	*BaseStateHandler
	proposalProcessor ProposalProcessor
	suffrage          Suffrage
	initBallotTimer   *localtime.CallbackTimer
	acceptBallotTimer *localtime.CallbackTimer
}

func NewConsensusStateConsensusHandler(
	localState *LocalState,
	proposalProcessor ProposalProcessor,
	suffrage Suffrage,
) (*ConsensusStateConsensusHandler, error) {
	if lastBlock := localState.LastBlock(); lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	cs := &ConsensusStateConsensusHandler{
		BaseStateHandler:  NewBaseStateHandler(localState, ConsensusStateConsensus),
		proposalProcessor: proposalProcessor,
		suffrage:          suffrage,
	}
	cs.BaseStateHandler.Logger = logging.NewLogger(func(c zerolog.Context) zerolog.Context {
		return c.Str("module", "consensus-state-consensus-handler")
	})

	return cs, nil
}

func (cs *ConsensusStateConsensusHandler) SetLogger(l zerolog.Logger) *logging.Logger {
	if cs.initBallotTimer != nil {
		_ = cs.initBallotTimer.SetLogger(l)
	}

	return cs.Logger.SetLogger(l)
}

func (cs *ConsensusStateConsensusHandler) Activate(ctx ConsensusStateChangeContext) error {
	l := loggerWithConsensusStateChangeContext(ctx, cs.Log())
	l.Debug().Msg("activated")

	_ = cs.localState.SetLastINITVoteProof(ctx.VoteProof())

	if err := cs.waitProposal(); err != nil {
		return err
	}
	// TODO find Proposal; Proposal can be received early.

	return nil
}

func (cs *ConsensusStateConsensusHandler) Deactivate(ctx ConsensusStateChangeContext) error {
	l := loggerWithConsensusStateChangeContext(ctx, cs.Log())
	l.Debug().Msg("deactivated")

	if err := cs.initializeTimer(); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateConsensusHandler) initializeTimer() error {
	cs.Lock()
	defer cs.Unlock()

	if cs.initBallotTimer != nil {
		if err := cs.initBallotTimer.Stop(); err != nil {
			return err
		}
		cs.initBallotTimer = nil
	}

	if cs.acceptBallotTimer != nil {
		if err := cs.acceptBallotTimer.Stop(); err != nil {
			return err
		}
		cs.acceptBallotTimer = nil
	}

	return nil
}

func (cs *ConsensusStateConsensusHandler) waitProposal() error {
	if err := cs.initializeTimer(); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()

	var calledCount int64
	nt, err := localtime.NewCallbackTimer(
		"consensus-next-round-if-waiting-proposal-timeout",
		func() (bool, error) {
			vp := cs.localState.LastINITVoteProof()

			l := loggerWithVoteProof(vp, cs.Log())
			if !vp.IsFinished() { // NOTE check VoteProof is empty
				l.Error().Err(xerrors.Errorf("invalid VoteProof found in LocalState"))
				return true, nil
			}

			round := vp.Round() + 1

			l.Debug().Msg("timeout; waiting Proposal; trying to move next round")

			ib, err := NewINITBallotV0FromLocalState(cs.localState, round, nil)
			if err != nil {
				l.Error().Err(err).Msg("failed to move next round; will keep trying")
				return true, nil
			}

			cs.BroadcastSeal(ib, nil)

			return true, nil
		},
		0,
		func() time.Duration {
			defer atomic.AddInt64(&calledCount, 1)

			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast INIT Ballot.
			if atomic.LoadInt64(&calledCount) < 1 {
				return cs.localState.Policy().TimeoutWaitingProposal()
			}

			return cs.localState.Policy().IntervalBroadcastingINITBallot()
		},
	)
	if err != nil {
		return err
	}

	cs.initBallotTimer = nt
	cs.initBallotTimer.SetLogger(*cs.Log())

	return cs.initBallotTimer.Start()
}

func (cs *ConsensusStateConsensusHandler) NewSeal(sl seal.Seal) error {
	switch t := sl.(type) {
	case Proposal:
		return cs.handleProposal(t)
	default:
		return nil
	}
}

func (cs *ConsensusStateConsensusHandler) NewVoteProof(vp VoteProof) error {
	l := loggerWithVoteProof(vp, cs.Log())

	l.Debug().Msg("VoteProof received")

	switch vp.Stage() {
	case StageACCEPT:
		_ = cs.localState.SetLastACCEPTVoteProof(vp)

		if err := cs.storeNewBlock(vp); err != nil {
			return err
		}

		return cs.keepBroadcastingINITBallotForNextBlock()
	case StageINIT:
		_ = cs.localState.SetLastINITVoteProof(vp)

		return cs.handleINITVoteProof(vp)
	default:
		err := xerrors.Errorf("invalid VoteProof received")

		l.Error().Err(err).Send()

		return err
	}
}

func (cs *ConsensusStateConsensusHandler) handleINITVoteProof(vp VoteProof) error {
	l := loggerWithVoteProof(vp, cs.Log())

	switch d := (vp.Height() - cs.localState.LastBlock().Height() + 1); {
	case d < 0: // old VoteProof
		// TODO check it's previousBlock and previousRound is matched with local.
		l.Debug().Msg("old VoteProof received; ignore it")
		return nil
	case d > 0: // far from local; moves to syncing
		if err := cs.ChangeState(ConsensusStateSyncing, vp); err != nil {
			return err
		}

		return nil
	default: // expected VoteProof
		return cs.waitProposal()
	}
}

func (cs *ConsensusStateConsensusHandler) keepBroadcastingINITBallotForNextBlock() error {
	if err := cs.initializeTimer(); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()

	nt, err := localtime.NewCallbackTimer(
		"consensus-broadcasting-init-ballot-for-next-block",
		func() (bool, error) {
			vp := cs.localState.LastINITVoteProof()

			l := loggerWithVoteProof(vp, cs.Log())
			if !vp.IsFinished() { // NOTE check VoteProof is empty
				l.Error().Err(xerrors.Errorf("invalid VoteProof found in LocalState"))
				return true, nil
			}

			l.Debug().Msg("trying to broadcast INIT Ballot for next block")

			ib, err := NewINITBallotV0FromLocalState(cs.localState, Round(0), nil)
			if err != nil {
				l.Error().Err(err).Msg("trying to broadcast INIT Ballot for next block; will keep trying")
				return true, nil
			}

			cs.BroadcastSeal(ib, nil)

			return true, nil
		},
		cs.localState.Policy().IntervalBroadcastingINITBallot(),
		nil,
	)
	if err != nil {
		return err
	}

	cs.initBallotTimer = nt
	cs.initBallotTimer.SetLogger(*cs.Log())

	return cs.initBallotTimer.Start()
}

func (cs *ConsensusStateConsensusHandler) storeNewBlock(vp VoteProof) error {
	// TODO store processed new block of Proposal
	fmt.Println(">", vp)
	return nil
}

func (cs *ConsensusStateConsensusHandler) handleProposal(proposal Proposal) error {
	l := loggerWithBallot(proposal, cs.Log())

	l.Debug().Msg("got proposal")

	if err := cs.initializeTimer(); err != nil {
		return err
	}

	newBlock, err := cs.proposalProcessor.Process(proposal)
	if err != nil {
		return err
	} else if newBlock == nil {
		return xerrors.Errorf("failed to process Proposal; empty Block returned")
	}

	acting := cs.suffrage.Acting(proposal.Height(), proposal.Round())
	isActing := acting.Exists(cs.localState.Node().Address())
	l.Debug().
		Object("acting_suffrag", acting).
		Bool("is_acting", isActing).
		Msg("node is in acting suffrage?")
	if isActing {
		// TODO broadcast SIGN Ballot if local node is in acting suffrage.
		if err := cs.readyToSIGNBallot(proposal, newBlock); err != nil {
			return err
		}
	}

	return cs.readyToACCEPTBallot(proposal, newBlock)
}

func (cs *ConsensusStateConsensusHandler) readyToSIGNBallot(proposal Proposal, newBlock Block) error {
	// NOTE not like broadcasting ACCEPT Ballot, SIGN Ballot will be broadcasted
	// withtout waiting.

	sb, err := NewSIGNBallotV0FromLocalState(cs.localState, proposal.Round(), newBlock, nil)
	if err != nil {
		cs.Log().Error().Err(err).Msg("failed to create SIGNBallot")
		return err
	}

	cs.BroadcastSeal(sb, nil)

	loggerWithBallot(sb, cs.Log()).Debug().Msg("SIGNBallot was broadcasted")

	return nil
}

func (cs *ConsensusStateConsensusHandler) readyToACCEPTBallot(proposal Proposal, newBlock Block) error {
	// TODO if not in acting suffrage, broadcast ACCEPT Ballot after interval.
	// TODO wait until the given interval.

	var calledCount int64
	nt, err := localtime.NewCallbackTimer(
		"consensus-broadcasting-accept-ballot",
		func() (bool, error) {
			// TODO ACCEPTBallot should include the received SIGNBallots.
			ab, err := NewACCEPTBallotV0FromLocalState(cs.localState, proposal.Round(), newBlock, nil)
			if err != nil {
				cs.Log().Error().Err(err).Msg("failed to create ACCEPTBallot; will keep trying")
				return true, nil
			}

			cs.BroadcastSeal(ab, nil)

			loggerWithBallot(ab, cs.Log()).Debug().Msg("ACCEPTBallot was broadcasted")

			return true, nil
		},
		0,
		func() time.Duration {
			defer atomic.AddInt64(&calledCount, 1)

			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast ACCEPT Ballot.
			if atomic.LoadInt64(&calledCount) < 1 {
				return cs.localState.Policy().WaitBroadcastingACCEPTBallot()
			}

			return cs.localState.Policy().IntervalBroadcastingACCEPTBallot()
		},
	)
	if err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()

	cs.acceptBallotTimer = nt
	cs.acceptBallotTimer.SetLogger(*cs.Log())

	return cs.acceptBallotTimer.Start()
}
