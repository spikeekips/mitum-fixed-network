package isaac

import (
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
	proposalMaker     *ProposalMaker
	ballotTimer       *localtime.CallbackTimer
}

func NewConsensusStateConsensusHandler(
	localState *LocalState,
	proposalProcessor ProposalProcessor,
	suffrage Suffrage,
	proposalMaker *ProposalMaker,
) (*ConsensusStateConsensusHandler, error) {
	if lastBlock := localState.LastBlock(); lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	cs := &ConsensusStateConsensusHandler{
		BaseStateHandler:  NewBaseStateHandler(localState, ConsensusStateConsensus),
		proposalProcessor: proposalProcessor,
		suffrage:          suffrage,
		proposalMaker:     proposalMaker,
	}
	cs.BaseStateHandler.Logger = logging.NewLogger(func(c zerolog.Context) zerolog.Context {
		return c.Str("module", "consensus-state-consensus-handler")
	})

	return cs, nil
}

func (cs *ConsensusStateConsensusHandler) SetLogger(l zerolog.Logger) *logging.Logger {
	if cs.ballotTimer != nil {
		_ = cs.ballotTimer.SetLogger(l)
	}

	return cs.Logger.SetLogger(l)
}

func (cs *ConsensusStateConsensusHandler) Activate(ctx ConsensusStateChangeContext) error {
	if ctx.VoteProof() == nil {
		return xerrors.Errorf("consensus handler got empty VoteProof")
	} else if err := ctx.VoteProof().IsValid(nil); err != nil {
		return xerrors.Errorf("consensus handler got invalid VoteProof: %w", err)
	}

	_ = cs.localState.SetLastINITVoteProof(ctx.VoteProof())

	l := loggerWithConsensusStateChangeContext(ctx, cs.Log())

	go func() {
		if err := cs.waitProposal(ctx.VoteProof()); err != nil {
			l.Error().Err(err).Send()
		}
	}()

	l.Debug().Msg("activated")

	return nil
}

func (cs *ConsensusStateConsensusHandler) Deactivate(ctx ConsensusStateChangeContext) error {
	l := loggerWithConsensusStateChangeContext(ctx, cs.Log())

	if err := cs.stopBallotTimer(); err != nil {
		return err
	}

	l.Debug().Msg("deactivated")

	return nil
}

func (cs *ConsensusStateConsensusHandler) startBallotTimer(timer *localtime.CallbackTimer) error {
	if err := cs.stopBallotTimer(); err != nil {
		return err
	}

	cs.Lock()
	defer cs.Unlock()

	timer.SetLogger(*cs.Log())

	cs.ballotTimer = timer

	if err := cs.ballotTimer.Start(); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateConsensusHandler) stopBallotTimer() error {
	cs.Lock()
	defer cs.Unlock()

	if cs.ballotTimer != nil {
		if err := cs.ballotTimer.Stop(); err != nil {
			return err
		}
		cs.ballotTimer = nil
	}

	return nil
}

func (cs *ConsensusStateConsensusHandler) waitProposal(vp VoteProof) error {
	cs.Log().Debug().Msg("waiting proposal")

	if proposed, err := cs.proposal(vp); err != nil {
		return err
	} else if proposed {
		return nil
	}

	var calledCount int64
	timer, err := localtime.NewCallbackTimer(
		"consensus-next-round-if-waiting-proposal-timeout",
		func() (bool, error) {
			vp := cs.localState.LastINITVoteProof()

			l := loggerWithVoteProof(vp, cs.Log())

			round := vp.Round() + 1

			l.Debug().
				Dur("timeout", cs.localState.Policy().TimeoutWaitingProposal()).
				Msg("timeout; waiting Proposal; trying to move next round")

			ib, err := NewINITBallotV0FromLocalState(cs.localState, round, nil)
			if err != nil {
				l.Error().Err(err).Msg("failed to move next round; will keep trying")
				return true, nil
			}

			cs.BroadcastSeal(ib)

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

	return cs.startBallotTimer(timer)
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
	if err := cs.stopBallotTimer(); err != nil {
		return err
	}

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
	l := loggerWithLocalState(cs.localState, loggerWithVoteProof(vp, cs.Log()))

	switch d := (vp.Height() - (cs.localState.LastBlock().Height() + 1)); {
	case d < 0: // old VoteProof
		// TODO check it's previousBlock and previousRound is matched with local.
		l.Debug().Msg("old VoteProof received; ignore it")
		return nil
	case d > 0: // far from local; moves to syncing
		l.Debug().Msg("higher VoteProof received; moves to sync")

		return cs.ChangeState(ConsensusStateSyncing, vp)
	default: // expected VoteProof
		l.Debug().Msg("expected VoteProof received; will wait Proposal")
		return cs.waitProposal(vp)
	}
}

func (cs *ConsensusStateConsensusHandler) keepBroadcastingINITBallotForNextBlock() error {
	timer, err := localtime.NewCallbackTimer(
		"consensus-broadcasting-init-ballot-for-next-block",
		func() (bool, error) {
			vp := cs.localState.LastINITVoteProof()

			l := loggerWithVoteProof(vp, cs.Log())

			l.Debug().Msg("trying to broadcast INIT Ballot for next block")

			ib, err := NewINITBallotV0FromLocalState(cs.localState, Round(0), nil)
			if err != nil {
				l.Error().Err(err).Msg("trying to broadcast INIT Ballot for next block; will keep trying")
				return true, nil
			}

			cs.BroadcastSeal(ib)

			return true, nil
		},
		cs.localState.Policy().IntervalBroadcastingINITBallot(),
		nil,
	)
	if err != nil {
		return err
	}

	return cs.startBallotTimer(timer)
}

func (cs *ConsensusStateConsensusHandler) storeNewBlock(vp VoteProof) error {
	fact, ok := vp.Majority().(ACCEPTBallotFact)
	if !ok {
		return xerrors.Errorf("needs ACCEPTBallotFact: fact=%T", vp.Majority())
	}

	l := loggerWithVoteProof(vp, cs.Log()).With().
		Str("proposal", fact.Proposal().String()).
		Str("new_block", fact.NewBlock().String()).
		Logger()

	l.Debug().Msg("trying to store new block")

	newBlock, err := cs.proposalProcessor.Process(fact.Proposal(), nil)
	if err != nil {
		return err
	}

	if newBlock == nil {
		err := xerrors.Errorf("failed to process Proposal; empty Block returned")
		l.Error().Err(err).Send()

		return err
	}

	if !fact.NewBlock().Equal(newBlock.Hash()) {
		err := xerrors.Errorf(
			"processed new block does not match; fact=%s processed=%s",
			fact.NewBlock(),
			newBlock.Hash(),
		)
		l.Error().Err(err).Send()

		return err
	}

	_ = cs.localState.SetLastBlock(newBlock)

	l.Info().Msg("new block stored")

	return nil
}

func (cs *ConsensusStateConsensusHandler) handleProposal(proposal Proposal) error {
	l := loggerWithBallot(proposal, cs.Log())

	l.Debug().Msg("got proposal")

	newBlock, err := cs.proposalProcessor.Process(proposal.Hash(), nil)
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

	cs.BroadcastSeal(sb)

	loggerWithBallot(sb, cs.Log()).Debug().Msg("SIGNBallot was broadcasted")

	return nil
}

func (cs *ConsensusStateConsensusHandler) readyToACCEPTBallot(proposal Proposal, newBlock Block) error {
	// TODO if not in acting suffrage, broadcast ACCEPT Ballot after interval.
	// TODO wait until the given interval.

	var calledCount int64
	timer, err := localtime.NewCallbackTimer(
		"consensus-broadcasting-accept-ballot",
		func() (bool, error) {
			// TODO ACCEPTBallot should include the received SIGN Ballots.

			ab, err := NewACCEPTBallotV0FromLocalState(cs.localState, proposal.Round(), newBlock, nil)
			if err != nil {
				cs.Log().Error().Err(err).Msg("failed to create ACCEPTBallot; will keep trying")
				return true, nil
			}

			l := loggerWithBallot(ab, cs.Log())
			cs.BroadcastSeal(ab)

			l.Debug().Msg("ACCEPTBallot was broadcasted")

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

	return cs.startBallotTimer(timer)
}

func (cs *ConsensusStateConsensusHandler) proposal(vp VoteProof) (bool, error) {
	// TODO find Proposal; Proposal can be received early.
	l := loggerWithVoteProof(vp, cs.Log())

	l.Debug().Msg("prepare to broadcast Proposal")
	isProposer := cs.suffrage.IsProposer(vp.Height(), vp.Round(), cs.localState.Node().Address())
	l.Debug().
		Object("acting_suffrag", cs.suffrage.Acting(vp.Height(), vp.Round())).
		Bool("is_acting", cs.suffrage.IsActing(vp.Height(), vp.Round(), cs.localState.Node().Address())).
		Bool("is_proposer", isProposer).
		Msg("node is proposer?")

	if !isProposer {
		return false, nil
	}

	proposal, err := cs.proposalMaker.Proposal(vp.Round(), nil)
	if err != nil {
		return false, err
	}

	l.Debug().Msg("trying to broadcast Proposal")

	cs.BroadcastSeal(proposal)

	return true, nil
}
