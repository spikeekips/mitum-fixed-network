package mitum

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"golang.org/x/xerrors"
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
	initBallotTimer *localtime.CallbackTimer
}

func NewConsensusStateConsensusHandler(
	localState *LocalState,
) (*ConsensusStateConsensusHandler, error) {
	cs := &ConsensusStateConsensusHandler{
		BaseStateHandler: NewBaseStateHandler(localState, ConsensusStateConsensus),
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
	log := loggerWithConsensusStateChangeContext(ctx, cs.Log())
	log.Debug().Msg("activated")

	_ = cs.localState.SetLastINITVoteProof(ctx.VoteProof())

	if err := cs.waitProposal(); err != nil {
		return err
	}

	return nil
}

func (cs *ConsensusStateConsensusHandler) Deactivate(ctx ConsensusStateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	log := loggerWithConsensusStateChangeContext(ctx, cs.Log())
	log.Debug().Msg("deactivated")

	if cs.initBallotTimer != nil {
		if err := cs.initBallotTimer.Stop(); err != nil {
			return err
		}
		cs.initBallotTimer = nil
	}

	return nil
}

func (cs *ConsensusStateConsensusHandler) waitProposal() error {
	if cs.initBallotTimer != nil {
		if err := cs.initBallotTimer.Stop(); err != nil {
			return err
		}
	}

	cs.Lock()
	defer cs.Unlock()

	var calledCount int64
	nt, err := localtime.NewCallbackTimer(
		"consensus-next-round-if-waiting-proposal-timeout",
		func() (bool, error) {
			vp := cs.localState.LastINITVoteProof()

			log := loggerWithVoteProof(vp, cs.Log())
			if !vp.IsFinished() { // NOTE check VoteProof is empty
				log.Error().Err(xerrors.Errorf("invalid VoteProof found in LocalState"))
				return true, nil
			}

			round := vp.Round() + 1

			log.Debug().Msg("timeout; waiting Proposal; trying to move next round")

			ib, err := NewINITBallotV0FromLocalState(cs.localState, round, nil)
			if err != nil {
				log.Error().Err(err).Msg("failed to move next round; will keep trying")
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

			// TODO this duration also should be managed by LocalState
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

func (cs *ConsensusStateConsensusHandler) handleProposal(proposal Proposal) error {
	log := cs.Log().With().
		Str("proposal_hash", proposal.Hash().String()).
		Int64("proposal_height", proposal.Height().Int64()).
		Uint64("proposal_round", proposal.Round().Uint64()).
		Logger()

	log.Debug().Msg("got proposal")

	// TODO check Proposal is proper
	// TODO process Proposal
	// TODO broadcast ACCEPT Ballot

	return nil
}

func (cs *ConsensusStateConsensusHandler) NewVoteProof(vp VoteProof) error {
	log := loggerWithVoteProof(vp, cs.Log())

	log.Debug().Msg("VoteProof received")

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

		log.Error().Err(err).Send()

		return err
	}
}

func (cs *ConsensusStateConsensusHandler) handleINITVoteProof(vp VoteProof) error {
	log := loggerWithVoteProof(vp, cs.Log())

	switch d := (vp.Height() - cs.localState.LastBlockHeight() + 1); {
	case d < 0: // old VoteProof
		// TODO check it's previousBlock and previousRound is matched with local.
		log.Debug().Msg("old VoteProof received; ignore it")
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
	if cs.initBallotTimer != nil {
		if err := cs.initBallotTimer.Stop(); err != nil {
			return err
		}
	}

	cs.Lock()
	defer cs.Unlock()

	nt, err := localtime.NewCallbackTimer(
		"consensus-broadcasting-init-ballot-for-next-block",
		func() (bool, error) {
			vp := cs.localState.LastINITVoteProof()

			log := loggerWithVoteProof(vp, cs.Log())
			if !vp.IsFinished() { // NOTE check VoteProof is empty
				log.Error().Err(xerrors.Errorf("invalid VoteProof found in LocalState"))
				return true, nil
			}

			log.Debug().Msg("trying to broadcast INIT Ballot for next block")

			ib, err := NewINITBallotV0FromLocalState(cs.localState, Round(0), nil)
			if err != nil {
				log.Error().Err(err).Msg("trying to broadcast INIT Ballot for next block; will keep trying")
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
