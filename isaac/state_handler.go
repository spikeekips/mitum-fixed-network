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
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

const (
	TimerIDBroadcastingINITBallot   = "broadcasting-init-ballot"
	TimerIDBroadcastingACCEPTBallot = "broadcasting-accept-ballot"
	TimerIDBroadcastingProposal     = "broadcasting-proposal"
	TimerIDTimedoutMoveNextRound    = "timedout-move-to-next-round"
)

type StateHandler interface {
	State() base.State
	SetStateChan(chan<- StateChangeContext)
	SetSealChan(chan<- seal.Seal)
	Activate(StateChangeContext) error
	Deactivate(StateChangeContext) error
	// NewSeal receives Seal.
	NewSeal(seal.Seal) error
	// NewVoteproof receives the finished Voteproof.
	NewVoteproof(base.Voteproof) error
}

type StateChangeContext struct {
	fromState base.State
	toState   base.State
	voteproof base.Voteproof
	ballot    ballot.Ballot
}

func NewStateChangeContext(from, to base.State, voteproof base.Voteproof, blt ballot.Ballot) StateChangeContext {
	return StateChangeContext{
		fromState: from,
		toState:   to,
		voteproof: voteproof,
		ballot:    blt,
	}
}

func (csc StateChangeContext) From() base.State {
	return csc.fromState
}

func (csc StateChangeContext) To() base.State {
	return csc.toState
}

func (csc StateChangeContext) Voteproof() base.Voteproof {
	return csc.voteproof
}

func (csc StateChangeContext) Ballot() ballot.Ballot {
	return csc.ballot
}

func (csc StateChangeContext) MarshalLog(key string, e logging.Emitter, _ bool) logging.Emitter {
	return e.Dict(key, logging.Dict().
		Hinted("from_state", csc.From()).
		Hinted("to_state", csc.To()),
	)
}

type BaseStateHandler struct {
	sync.RWMutex
	*logging.Logging
	localstate        *Localstate
	proposalProcessor ProposalProcessor
	state             base.State
	stateChan         chan<- StateChangeContext
	sealChan          chan<- seal.Seal
	timers            *localtime.Timers
}

func NewBaseStateHandler(
	localstate *Localstate, proposalProcessor ProposalProcessor, state base.State,
) *BaseStateHandler {
	return &BaseStateHandler{
		localstate:        localstate,
		proposalProcessor: proposalProcessor,
		state:             state,
	}
}

func (bs *BaseStateHandler) State() base.State {
	return bs.state
}

func (bs *BaseStateHandler) SetStateChan(stateChan chan<- StateChangeContext) {
	bs.stateChan = stateChan
}

func (bs *BaseStateHandler) SetSealChan(sealChan chan<- seal.Seal) {
	bs.sealChan = sealChan
}

func (bs *BaseStateHandler) ChangeState(newState base.State, voteproof base.Voteproof, blt ballot.Ballot) error {
	if bs.stateChan == nil {
		return nil
	}

	if err := newState.IsValid(nil); err != nil {
		return err
	} else if newState == bs.state {
		return xerrors.Errorf("can not change state to same state; current=%s new=%s", bs.state, newState)
	}

	go func() {
		bs.stateChan <- NewStateChangeContext(bs.state, newState, voteproof, blt)
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

func (bs *BaseStateHandler) StoreNewBlock(blockStorage storage.BlockStorage) error {
	if err := blockStorage.Commit(); err != nil {
		return err
	}

	return nil
}

func (bs *BaseStateHandler) StoreNewBlockByVoteproof(acceptVoteproof base.Voteproof) error {
	if bs.proposalProcessor == nil {
		bs.Log().Debug().Msg("this state not support store new block")

		return nil
	}

	fact, ok := acceptVoteproof.Majority().(ballot.ACCEPTBallotFact)
	if !ok {
		return xerrors.Errorf("needs ACCEPTBallotFact: fact=%T", acceptVoteproof.Majority())
	}

	l := loggerWithVoteproof(
		acceptVoteproof,
		bs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
			return ctx.Hinted("proposal", fact.Proposal()).
				Hinted("new_block", fact.NewBlock())
		}),
	)

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

	l.Info().Dict("block", logging.Dict().
		Hinted("proposal", blockStorage.Block().Proposal()).
		Hinted("hash", blockStorage.Block().Hash()).
		Hinted("height", blockStorage.Block().Height()).
		Hinted("round", blockStorage.Block().Round()),
	).
		Msg("new block stored")

	return nil
}

func (bs *BaseStateHandler) TimerBroadcastingINITBallot(
	intervalFunc func() time.Duration,
	roundFunc func() base.Round,
) (*localtime.CallbackTimer, error) {
	var baseBallot ballot.INITBallotV0

	round := roundFunc()
	if round == 0 {
		if b, err := NewINITBallotV0Round0(bs.localstate.Storage(), bs.localstate.Node().Address()); err != nil {
			return nil, err
		} else {
			baseBallot = b
		}
	} else {
		if b, err := NewINITBallotV0WithVoteproof(
			bs.localstate.Storage(),
			bs.localstate.Node().Address(),
			round,
			bs.localstate.LastINITVoteproof(),
		); err != nil {
			return nil, err
		} else {
			baseBallot = b
		}
	}

	return localtime.NewCallbackTimer(
		TimerIDBroadcastingINITBallot,
		func() (bool, error) {
			ib := baseBallot
			if err := SignSeal(&ib, bs.localstate); err != nil {
				bs.Log().Error().Err(err).Msg("failed to re-sign INITBallot, but will keep trying")

				return true, nil
			}

			bs.BroadcastSeal(ib)

			return true, nil
		},
		0,
		intervalFunc,
	)
}

func (bs *BaseStateHandler) TimerBroadcastingACCEPTBallot(newBlock block.Block) (*localtime.CallbackTimer, error) {
	baseBallot := NewACCEPTBallotV0(bs.localstate.Node().Address(), newBlock, bs.localstate.LastINITVoteproof())

	var called int64

	return localtime.NewCallbackTimer(
		TimerIDBroadcastingACCEPTBallot,
		func() (bool, error) {
			ab := baseBallot
			if err := SignSeal(&ab, bs.localstate); err != nil {
				bs.Log().Error().Err(err).Msg("failed to re-sign ACCEPTBallot, but will keep trying")

				return true, nil
			}

			bs.BroadcastSeal(ab)

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
	round base.Round,
) (*localtime.CallbackTimer, error) {
	var baseBallot ballot.INITBallotV0
	if round == 0 {
		if b, err := NewINITBallotV0Round0(bs.localstate.Storage(), bs.localstate.Node().Address()); err != nil {
			return nil, err
		} else {
			baseBallot = b
		}
	} else {
		if b, err := NewINITBallotV0WithVoteproof(
			bs.localstate.Storage(),
			bs.localstate.Node().Address(),
			round,
			bs.localstate.LastINITVoteproof(),
		); err != nil {
			return nil, err
		} else {
			baseBallot = b
		}
	}

	var called int64

	return localtime.NewCallbackTimer(
		TimerIDTimedoutMoveNextRound,
		func() (bool, error) {
			bs.Log().Debug().
				Dur("timeout", bs.localstate.Policy().TimeoutWaitingProposal()).
				Hinted("next_round", round).
				Msg("timeout; waiting Proposal; trying to move next round")

			if err := bs.timers.StopTimers([]string{TimerIDBroadcastingINITBallot}); err != nil {
				bs.Log().Error().Err(err).Str("timer", TimerIDBroadcastingINITBallot).Msg("failed to stop")
			}

			ib := baseBallot
			if err := SignSeal(&ib, bs.localstate); err != nil {
				bs.Log().Error().Err(err).Msg("failed to re-sign ACCEPTBallot, but will keep trying")

				return true, nil
			}

			bs.BroadcastSeal(ib)

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

func (bs *BaseStateHandler) TimerBroadcastingProposal(proposal ballot.Proposal) (*localtime.CallbackTimer, error) {
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
