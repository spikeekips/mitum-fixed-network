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
)

const (
	TimerIDBroadcastingINITBallot   = "broadcasting-init-ballot"
	TimerIDBroadcastingACCEPTBallot = "broadcasting-accept-ballot"
	TimerIDBroadcastingProposal     = "broadcasting-proposal"
	TimerIDTimedoutMoveNextRound    = "timedout-move-to-next-round"
)

type StateHandler interface {
	State() State
	SetStateChan(chan<- StateChangeContext)
	SetSealChan(chan<- seal.Seal)
	Activate(StateChangeContext) error
	Deactivate(StateChangeContext) error
	// NewSeal receives Seal.
	NewSeal(seal.Seal) error
	// NewVoteproof receives the finished Voteproof.
	NewVoteproof(Voteproof) error
}

type StateChangeContext struct {
	fromState State
	toState   State
	voteproof Voteproof
}

func NewStateChangeContext(from, to State, voteproof Voteproof) StateChangeContext {
	return StateChangeContext{
		fromState: from,
		toState:   to,
		voteproof: voteproof,
	}
}

func (csc StateChangeContext) From() State {
	return csc.fromState
}

func (csc StateChangeContext) To() State {
	return csc.toState
}

func (csc StateChangeContext) Voteproof() Voteproof {
	return csc.voteproof
}

type BaseStateHandler struct {
	sync.RWMutex
	*logging.Logger
	localstate        *Localstate
	proposalProcessor ProposalProcessor
	state             State
	stateChan         chan<- StateChangeContext
	sealChan          chan<- seal.Seal
	timers            *localtime.Timers
}

func NewBaseStateHandler(
	localstate *Localstate, proposalProcessor ProposalProcessor, state State,
) *BaseStateHandler {
	return &BaseStateHandler{
		localstate:        localstate,
		proposalProcessor: proposalProcessor,
		state:             state,
	}
}

func (bs *BaseStateHandler) State() State {
	return bs.state
}

func (bs *BaseStateHandler) SetStateChan(stateChan chan<- StateChangeContext) {
	bs.stateChan = stateChan
}

func (bs *BaseStateHandler) SetSealChan(sealChan chan<- seal.Seal) {
	bs.sealChan = sealChan
}

func (bs *BaseStateHandler) ChangeState(newState State, voteproof Voteproof) error {
	if bs.stateChan == nil {
		return nil
	}

	if err := newState.IsValid(nil); err != nil {
		return err
	} else if newState == bs.state {
		return xerrors.Errorf("can not change state to same state; current=%s new=%s", bs.state, newState)
	}

	go func() {
		bs.stateChan <- NewStateChangeContext(bs.state, newState, voteproof)
	}()

	return nil
}

func (bs *BaseStateHandler) BroadcastSeal(sl seal.Seal) {
	if bs.sealChan == nil {
		return
	}

	go func() {
		bs.sealChan <- sl
	}()
}

func (bs *BaseStateHandler) StoreNewBlock(blockStorage BlockStorage) error {
	if err := blockStorage.Commit(); err != nil {
		return err
	}

	_ = bs.localstate.SetLastBlock(blockStorage.Block())

	return nil
}

func (bs *BaseStateHandler) StoreNewBlockByVoteproof(acceptVoteproof Voteproof) error {
	fact, ok := acceptVoteproof.Majority().(ACCEPTBallotFact)
	if !ok {
		return xerrors.Errorf("needs ACCEPTBallotFact: fact=%T", acceptVoteproof.Majority())
	}

	l := loggerWithVoteproof(acceptVoteproof, bs.Log()).With().
		Str("proposal", fact.Proposal().String()).
		Str("new_block", fact.NewBlock().String()).
		Logger()

	_ = bs.localstate.SetLastACCEPTVoteproof(acceptVoteproof)

	l.Debug().Msg("trying to store new block")

	blockStorage, err := bs.proposalProcessor.ProcessACCEPT(fact.Proposal(), acceptVoteproof)
	if err != nil {
		return err
	}

	if blockStorage.Block() == nil {
		err := xerrors.Errorf("failed to process Proposal; empty Block returned")
		l.Error().Err(err).Msg("failed to store new block")

		return err
	}

	if !fact.NewBlock().Equal(blockStorage.Block().Hash()) {
		err := xerrors.Errorf(
			"processed new block does not match; fact=%s processed=%s",
			fact.NewBlock(),
			blockStorage.Block().Hash(),
		)
		l.Error().Err(err).Msg("failed to store new block")

		return err
	}

	if err := bs.StoreNewBlock(blockStorage); err != nil {
		l.Error().Err(err).Msg("failed to store new block")
		return err
	}

	l.Info().Dict("block", zerolog.Dict().
		Str("proposal", blockStorage.Block().Proposal().String()).
		Str("hash", blockStorage.Block().Hash().String()).
		Int64("height", blockStorage.Block().Height().Int64()).
		Uint64("round", blockStorage.Block().Round().Uint64()),
	).
		Msg("new block stored")

	return nil
}

func (bs *BaseStateHandler) TimerBroadcastingINITBallot(
	intervalFunc func() time.Duration,
	roundFunc func() Round,
) (*localtime.CallbackTimer, error) {
	return localtime.NewCallbackTimer(
		TimerIDBroadcastingINITBallot,
		func() (bool, error) {
			if ib, err := NewINITBallotV0FromLocalstate(bs.localstate, roundFunc()); err != nil {
				bs.Log().Error().Err(err).Msg("failed to broadcast INIT ballot; will keep trying")
				return true, nil
			} else {
				bs.BroadcastSeal(ib)
			}

			return true, nil
		},
		0,
		intervalFunc,
	)
}

func (bs *BaseStateHandler) TimerBroadcastingACCEPTBallot(newBlock Block) (*localtime.CallbackTimer, error) {
	var called int64

	return localtime.NewCallbackTimer(
		TimerIDBroadcastingACCEPTBallot,
		func() (bool, error) {
			if ab, err := NewACCEPTBallotV0FromLocalstate(bs.localstate, newBlock.Round(), newBlock); err != nil {
				bs.Log().Error().Err(err).Msg("failed to create ACCEPTBallot, but will keep trying")
				return true, nil
			} else {
				bs.BroadcastSeal(ab)
			}

			return true, nil
		},
		0,
		func() time.Duration {
			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast ACCEPT Ballot.
			if atomic.LoadInt64(&called) < 1 {
				atomic.AddInt64(&called, 1)
				return bs.localstate.Policy().WaitBroadcastingACCEPTBallot()
			}

			return bs.localstate.Policy().IntervalBroadcastingACCEPTBallot()
		},
	)
}

func (bs *BaseStateHandler) TimerTimedoutMoveNextRound(
	round Round,
) (*localtime.CallbackTimer, error) {
	var called int64

	return localtime.NewCallbackTimer(
		TimerIDTimedoutMoveNextRound,
		func() (bool, error) {
			bs.Log().Debug().
				Dur("timeout", bs.localstate.Policy().TimeoutWaitingProposal()).
				Uint64("next_round", round.Uint64()).
				Msg("timeout; waiting Proposal; trying to move next round")

			if err := bs.timers.StopTimers([]string{TimerIDBroadcastingINITBallot}); err != nil {
				bs.Log().Error().Err(err).Str("timer", TimerIDBroadcastingINITBallot).Msg("failed to stop")
			}

			if ib, err := NewINITBallotV0FromLocalstate(bs.localstate, round); err != nil {
				bs.Log().Error().Err(err).Msg("failed to move next round; will keep trying")

				return true, nil
			} else {
				bs.BroadcastSeal(ib)
			}

			return true, nil
		},
		0,
		func() time.Duration {
			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast INIT Ballot.
			if atomic.LoadInt64(&called) < 1 {
				atomic.AddInt64(&called, 1)
				return bs.localstate.Policy().TimeoutWaitingProposal()
			}

			return bs.localstate.Policy().IntervalBroadcastingINITBallot()
		},
	)
}

func (bs *BaseStateHandler) TimerBroadcastingProposal(proposal Proposal) (*localtime.CallbackTimer, error) {
	var called int64

	return localtime.NewCallbackTimer(
		TimerIDBroadcastingProposal,
		func() (bool, error) {
			bs.BroadcastSeal(proposal)

			return true, nil
		},
		0,
		func() time.Duration {
			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast.
			if atomic.LoadInt64(&called) < 1 {
				atomic.AddInt64(&called, 1)
				return time.Nanosecond
			}

			return bs.localstate.Policy().IntervalBroadcastingProposal()
		},
	)
}
