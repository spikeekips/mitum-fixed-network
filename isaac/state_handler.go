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
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

const (
	TimerIDBroadcastingINITBallot     = "broadcasting-init-ballot"
	TimerIDBroadcastingACCEPTBallot   = "broadcasting-accept-ballot"
	TimerIDBroadcastingProposal       = "broadcasting-proposal"
	TimerIDTimedoutMoveNextRound      = "timedout-move-to-next-round"
	TimerIDTimedoutProcessingProposal = "timedout-processing-proposal"
	TimerIDNodeInfo                   = "node-info"
)

type StateHandler interface {
	State() base.State
	SetStateChan(chan<- *StateChangeContext)
	SetSealChan(chan<- seal.Seal)
	SetVoteproofChan(chan<- base.Voteproof)
	Activate(*StateChangeContext) error
	Deactivate(*StateChangeContext) error
	// NewSeal receives Seal.
	NewSeal(seal.Seal) error
	// NewVoteproof receives the finished Voteproof.
	NewVoteproof(base.Voteproof) error
	LastINITVoteproof() base.Voteproof
	SetLastINITVoteproof(base.Voteproof) error
}

type StateChangeContext struct {
	fromState base.State
	toState   base.State
	voteproof base.Voteproof
	ballot    ballot.Ballot
}

func NewStateChangeContext(from, to base.State, voteproof base.Voteproof, blt ballot.Ballot) *StateChangeContext {
	return &StateChangeContext{
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
		Hinted("from", csc.From()).
		Hinted("to", csc.To()),
	)
}

type BaseStateHandler struct {
	sync.RWMutex
	*logging.Logging
	local          *Local
	pps            *prprocessor.Processors
	state          base.State
	activatedLock  sync.RWMutex
	activated      bool
	timers         *localtime.Timers
	livp           base.Voteproof
	stateChan      chan<- *StateChangeContext
	sealChan       chan<- seal.Seal
	voteproofChan  chan<- base.Voteproof
	whenBlockSaved func([]block.Block)
}

func NewBaseStateHandler(local *Local, pps *prprocessor.Processors, state base.State) *BaseStateHandler {
	return &BaseStateHandler{
		local:          local,
		pps:            pps,
		state:          state,
		whenBlockSaved: func([]block.Block) {},
	}
}

func (bs *BaseStateHandler) activate() {
	bs.activatedLock.Lock()
	defer bs.activatedLock.Unlock()

	bs.activated = true
}

func (bs *BaseStateHandler) deactivate() {
	bs.activatedLock.Lock()
	defer bs.activatedLock.Unlock()

	bs.activated = false
}

func (bs *BaseStateHandler) isActivated() bool {
	bs.activatedLock.RLock()
	defer bs.activatedLock.RUnlock()

	return bs.activated
}

func (bs *BaseStateHandler) State() base.State {
	return bs.state
}

func (bs *BaseStateHandler) SetStateChan(stateChan chan<- *StateChangeContext) {
	bs.stateChan = stateChan
}

func (bs *BaseStateHandler) SetSealChan(sealChan chan<- seal.Seal) {
	bs.sealChan = sealChan
}

func (bs *BaseStateHandler) SetVoteproofChan(ch chan<- base.Voteproof) {
	bs.voteproofChan = ch
}

func (bs *BaseStateHandler) ChangeState(newState base.State, voteproof base.Voteproof, blt ballot.Ballot) error {
	if !bs.isActivated() {
		return nil
	}

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
	bs.RLock()
	defer bs.RUnlock()

	if bs.sealChan == nil {
		return
	}

	go func() {
		bs.sealChan <- sl
	}()
}

func (bs *BaseStateHandler) VoteproofToStates(voteproof base.Voteproof) {
	bs.voteproofChan <- voteproof
}

func (bs *BaseStateHandler) StoreNewBlock(acceptVoteproof base.Voteproof) error {
	if !bs.isActivated() {
		return nil
	} else if bs.pps == nil {
		bs.Log().Debug().Msg("this state not support store new block")

		return nil
	}

	var fact ballot.ACCEPTBallotFact
	if f, ok := acceptVoteproof.Majority().(ballot.ACCEPTBallotFact); !ok {
		return xerrors.Errorf("needs ACCEPTBallotFact: fact=%T", acceptVoteproof.Majority())
	} else {
		fact = f
	}

	l := loggerWithVoteproof(acceptVoteproof, bs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("proposal_hash", fact.Proposal()).
			Dict("block", logging.Dict().
				Hinted("hash", fact.NewBlock()).Hinted("height", acceptVoteproof.Height()).Hinted("round", acceptVoteproof.Round()))
	}))

	if err := bs.storeNewBlock(fact, acceptVoteproof); err != nil {
		l.Error().Err(err).Msg("failed to store new block")

		return err
	}

	return nil
}

func (bs *BaseStateHandler) storeNewBlock(fact ballot.ACCEPTBallotFact, acceptVoteproof base.Voteproof) error {
	l := loggerWithVoteproof(acceptVoteproof, bs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("proposal_hash", fact.Proposal()).
			Dict("block", logging.Dict().
				Hinted("hash", fact.NewBlock()).Hinted("height", acceptVoteproof.Height()).Hinted("round", acceptVoteproof.Round()))
	}))
	l.Debug().Msg("trying to store new block")

	s := time.Now()

	timeout := bs.local.Policy().TimeoutWaitingProposal()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	l.Debug().Dur("timeout", timeout).Msg("trying to commit block")

	var newBlock block.Block
	if result := <-bs.pps.Save(ctx, fact.Proposal(), acceptVoteproof); result.Err != nil {
		return xerrors.Errorf("failed to process ACCEPT Voteproof: %w", result.Err)
	} else {
		newBlock = bs.pps.Current().Block()
		if newBlock == nil {
			err := xerrors.Errorf("failed to process Proposal; empty Block returned")
			l.Error().Err(err).Msg("failed to store new block")

			return err
		}
	}

	l.Info().Dur("elapsed", time.Since(s)).Msg("new block stored")

	bs.whenBlockSaved([]block.Block{newBlock})

	return nil
}

func (bs *BaseStateHandler) TimerBroadcastingINITBallot(
	intervalFunc func(int) time.Duration,
	round base.Round,
	voteproof base.Voteproof,
) (*localtime.CallbackTimer, error) {
	if !bs.isActivated() {
		return nil, nil
	}

	var baseBallot ballot.INITBallotV0

	if round == 0 {
		if b, err := NewINITBallotV0Round0(bs.local); err != nil {
			return nil, err
		} else {
			baseBallot = b
		}
	} else {
		if b, err := NewINITBallotV0WithVoteproof(bs.local.Node().Address(), voteproof); err != nil {
			return nil, err
		} else {
			baseBallot = b
		}
	}

	ct, err := localtime.NewCallbackTimer(
		TimerIDBroadcastingINITBallot,
		func(int) (bool, error) {
			if !bs.isActivated() {
				return false, nil
			}

			ib := baseBallot
			if err := SignSeal(&ib, bs.local); err != nil {
				bs.Log().Error().Err(err).Msg("failed to re-sign INITBallot, but will keep trying")

				return true, nil
			}

			bs.BroadcastSeal(ib)

			return true, nil
		},
		0,
	)

	if err != nil {
		return nil, err
	} else {
		return ct.SetInterval(intervalFunc), nil
	}
}

func (bs *BaseStateHandler) TimerBroadcastingACCEPTBallot(
	newBlock block.Block,
	voteproof base.Voteproof,
) (*localtime.CallbackTimer, error) {
	if !bs.isActivated() {
		return nil, nil
	}

	baseBallot := NewACCEPTBallotV0(bs.local.Node().Address(), newBlock, voteproof)

	ct, err := localtime.NewCallbackTimer(
		TimerIDBroadcastingACCEPTBallot,
		func(int) (bool, error) {
			if !bs.isActivated() {
				return false, nil
			}

			ab := baseBallot
			if err := SignSeal(&ab, bs.local); err != nil {
				bs.Log().Error().Err(err).Msg("failed to re-sign ACCEPTBallot, but will keep trying")

				return true, nil
			}

			bs.BroadcastSeal(ab)

			return true, nil
		},
		0,
	)

	if err != nil {
		return nil, err
	} else {
		return ct.SetInterval(func(i int) time.Duration {
			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast ACCEPT Ballot.
			if i < 1 {
				return bs.local.Policy().WaitBroadcastingACCEPTBallot()
			}

			return bs.local.Policy().IntervalBroadcastingACCEPTBallot()
		}), nil
	}
}

func (bs *BaseStateHandler) TimerTimedoutMoveNextRound(voteproof base.Voteproof) (*localtime.CallbackTimer, error) {
	if ct, err := bs.timerTimedoutMoveNextRound(
		voteproof,
		TimerIDTimedoutMoveNextRound,
		[]string{TimerIDBroadcastingINITBallot},
	); err != nil {
		return nil, err
	} else {
		return ct.SetInterval(func(i int) time.Duration {
			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast INIT Ballot.
			if i < 1 {
				return bs.local.Policy().TimeoutWaitingProposal()
			}

			return bs.local.Policy().IntervalBroadcastingINITBallot()
		}), nil
	}
}

func (bs *BaseStateHandler) TimerTimedoutProcessingProposal(
	voteproof base.Voteproof,
	timeout time.Duration,
) (*localtime.CallbackTimer, error) {
	if ct, err := bs.timerTimedoutMoveNextRound(
		voteproof,
		TimerIDTimedoutProcessingProposal,
		[]string{TimerIDBroadcastingINITBallot, TimerIDTimedoutMoveNextRound},
	); err != nil {
		return nil, err
	} else {
		return ct.SetInterval(func(i int) time.Duration {
			if i < 1 {
				return timeout
			}

			return bs.local.Policy().IntervalBroadcastingINITBallot()
		}), nil
	}
}

func (bs *BaseStateHandler) timerTimedoutMoveNextRound(
	voteproof base.Voteproof,
	timerid string,
	stopTimers []string,
) (*localtime.CallbackTimer, error) {
	if !bs.isActivated() {
		return nil, nil
	}

	var baseBallot ballot.INITBallotV0
	if b, err := NewINITBallotV0WithVoteproof(bs.local.Node().Address(), voteproof); err != nil {
		return nil, err
	} else {
		baseBallot = b
	}

	return localtime.NewCallbackTimer(
		timerid,
		func(int) (bool, error) {
			if !bs.isActivated() {
				return false, nil
			}

			bs.Log().Debug().
				Str("timer_id", timerid).
				Hinted("height", baseBallot.Height()).
				Hinted("next_round", baseBallot.Round()).
				Msg("timeout; trying to move next round")

			if err := bs.timers.StopTimers(stopTimers); err != nil {
				bs.Log().Error().Err(err).Strs("timers", stopTimers).Msg("failed to stop")
			}

			ib := baseBallot
			if err := SignSeal(&ib, bs.local); err != nil {
				bs.Log().Error().Err(err).Msg("failed to re-sign INITTBallot, but will keep trying")

				return true, nil
			}

			bs.BroadcastSeal(ib)

			return true, nil
		},
		0,
	)
}

func (bs *BaseStateHandler) TimerBroadcastingProposal(proposal ballot.Proposal) (*localtime.CallbackTimer, error) {
	if !bs.isActivated() {
		return nil, nil
	}

	ct, err := localtime.NewCallbackTimer(
		TimerIDBroadcastingProposal,
		func(int) (bool, error) {
			if !bs.isActivated() {
				return false, nil
			}

			bs.BroadcastSeal(proposal)

			return true, nil
		},
		0,
	)

	if err != nil {
		return nil, err
	} else {
		return ct.SetInterval(func(i int) time.Duration {
			// NOTE at 1st time, wait timeout duration, after then, periodically
			// broadcast.
			if i < 1 {
				return time.Nanosecond
			}

			return bs.local.Policy().IntervalBroadcastingProposal()
		}), nil
	}
}

func (bs *BaseStateHandler) LastINITVoteproof() base.Voteproof {
	bs.RLock()
	defer bs.RUnlock()

	return bs.livp
}

func (bs *BaseStateHandler) SetLastINITVoteproof(voteproof base.Voteproof) error {
	bs.Lock()
	defer bs.Unlock()

	if voteproof != nil && voteproof.Stage() != base.StageINIT {
		return xerrors.Errorf("invalid voteproof, %v for init", voteproof.Stage())
	}

	switch v := bs.livp; {
	case v == nil:
	case v.Height() > voteproof.Height():
		return xerrors.Errorf("lower height; %v > %v", v.Height(), voteproof.Height())
	case v.Height() == voteproof.Height():
		if v.Round() > voteproof.Round() {
			return xerrors.Errorf("same height, but lower round; %v, %v > %v", v.Height(), v.Round(), voteproof.Round())
		}
	}

	bs.livp = voteproof

	// NOTE cancel the previous processor

	return nil
}

func (bs *BaseStateHandler) WhenBlockSaved(callback func([]block.Block)) {
	bs.Lock()
	defer bs.Unlock()

	bs.whenBlockSaved = callback
}

func (bs *BaseStateHandler) findProposal(h valuehash.Hash) (ballot.Proposal, error) {
	if sl, found, err := bs.local.Storage().Seal(h); !found {
		return nil, storage.NotFoundError.Errorf("seal not found")
	} else if err != nil {
		return nil, err
	} else if pr, ok := sl.(ballot.Proposal); !ok {
		return nil, xerrors.Errorf("seal is not Proposal: %T", sl)
	} else {
		return pr, nil
	}
}
