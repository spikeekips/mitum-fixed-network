package isaac

import (
	"context"
	"sync"
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

const (
	TimerIDBroadcastingINITBallot   = "broadcasting-init-ballot"
	TimerIDBroadcastingACCEPTBallot = "broadcasting-accept-ballot"
	TimerIDBroadcastingProposal     = "broadcasting-proposal"
	TimerIDTimedoutMoveNextRound    = "timedout-move-to-next-round"
	TimerIDNodeInfo                 = "node-info"
)

type StateHandler interface {
	State() base.State
	SetStateChan(chan<- *StateChangeContext)
	SetSealChan(chan<- seal.Seal)
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
	local             *Local
	proposalProcessor ProposalProcessor
	state             base.State
	activatedLock     sync.RWMutex
	activated         bool
	timers            *localtime.Timers
	livp              base.Voteproof
	stateChan         chan<- *StateChangeContext
	sealChan          chan<- seal.Seal
	whenBlockSaved    func([]block.Block)
}

func NewBaseStateHandler(
	local *Local, proposalProcessor ProposalProcessor, state base.State,
) *BaseStateHandler {
	return &BaseStateHandler{
		local:             local,
		proposalProcessor: proposalProcessor,
		state:             state,
		whenBlockSaved:    func([]block.Block) {},
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

	if bs.proposalProcessor != nil {
		if err := bs.proposalProcessor.Cancel(); err != nil {
			bs.Log().Error().Err(err).Msg("failed to cancel proposal processor")
		}
	}
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
	if !bs.isActivated() {
		return
	}

	if bs.sealChan == nil {
		return
	}

	go func() {
		bs.sealChan <- sl
	}()
}

func (bs *BaseStateHandler) StoreNewBlock(acceptVoteproof base.Voteproof) error {
	if !bs.isActivated() {
		return nil
	} else if bs.proposalProcessor == nil {
		bs.Log().Debug().Msg("this state not support store new block")

		return nil
	}

	var fact ballot.ACCEPTBallotFact
	if f, ok := acceptVoteproof.Majority().(ballot.ACCEPTBallotFact); !ok {
		return xerrors.Errorf("needs ACCEPTBallotFact: fact=%T", acceptVoteproof.Majority())
	} else {
		fact = f
	}

	l := loggerWithVoteproofID(acceptVoteproof, bs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("proposal_hash", fact.Proposal()).
			Dict("block", logging.Dict().
				Hinted("hash", fact.NewBlock()).Hinted("height", acceptVoteproof.Height()).Hinted("round", acceptVoteproof.Round()))
	}))

	if err := util.Retry(3, time.Millisecond*200, func() error {
		if err := bs.storeNewBlock(fact, acceptVoteproof); err == nil {
			return nil
		} else {
			var ctx *StateToBeChangeError
			switch {
			case xerrors.As(err, &ctx):
				l.Error().Err(err).Msg("state will be moved with accept voteproof")

				return util.StopRetryingError.Wrap(err)
			case xerrors.Is(err, util.IgnoreError):
				return util.StopRetryingError.Wrap(err)
			default:
				l.Error().Err(err).Msg("something wrong to store accept voteproof; will retry")

				return err
			}
		}
	}); err != nil {
		l.Error().Err(err).Msg("failed to store new block after retrial")

		if e := bs.proposalProcessor.Cancel(); e != nil {
			return xerrors.Errorf(
				"failed to be store new block; and failed to be done ProposalProcessor: %w",
				e)
		}

		return err
	}

	return nil
}

func (bs *BaseStateHandler) storeNewBlock(fact ballot.ACCEPTBallotFact, acceptVoteproof base.Voteproof) error {
	l := loggerWithVoteproofID(acceptVoteproof, bs.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("proposal_hash", fact.Proposal()).
			Dict("block", logging.Dict().
				Hinted("hash", fact.NewBlock()).Hinted("height", acceptVoteproof.Height()).Hinted("round", acceptVoteproof.Round()))
	}))
	l.Debug().Msg("trying to store new block")

	var blockStorage storage.BlockStorage
	switch bs, err := bs.proposalProcessor.ProcessACCEPT(fact.Proposal(), acceptVoteproof); {
	case err != nil:
		return xerrors.Errorf("failed to process ACCEPT Voteproof: %w", err)
	case bs.Block() == nil:
		err := xerrors.Errorf("failed to process Proposal; empty Block returned")
		l.Error().Err(err).Msg("failed to store new block")

		return err
	default:
		blockStorage = bs
	}

	defer func() {
		_ = blockStorage.Close()
	}()

	var newBlock block.Block
	if blk := blockStorage.Block(); !fact.NewBlock().Equal(blk.Hash()) {
		err := xerrors.Errorf("processed new block does not match; fact=%s processed=%s",
			fact.NewBlock(), blk.Hash())
		l.Error().Err(err).Msg("failed to store new block; moves to sync")

		return NewStateToBeChangeError(base.StateSyncing, acceptVoteproof, nil, err)
	} else {
		newBlock = blk
	}

	s := time.Now()

	timeout := bs.local.Policy().TimeoutWaitingProposal()
	l.Debug().Dur("timeout", timeout).Msg("trying to commit block")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := blockStorage.Commit(ctx); err != nil {
		l.Error().Err(err).Msg("failed to store new block")

		return err
	}

	if err := bs.proposalProcessor.Done(fact.Proposal()); err != nil {
		return xerrors.Errorf("failed to be done ProposalProcessor: %w", err)
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
	if !bs.isActivated() {
		return nil, nil
	}

	var baseBallot ballot.INITBallotV0
	if b, err := NewINITBallotV0WithVoteproof(bs.local.Node().Address(), voteproof); err != nil {
		return nil, err
	} else {
		baseBallot = b
	}

	ct, err := localtime.NewCallbackTimer(
		TimerIDTimedoutMoveNextRound,
		func(int) (bool, error) {
			if !bs.isActivated() {
				return false, nil
			}

			bs.Log().Debug().
				Dur("timeout", bs.local.Policy().TimeoutWaitingProposal()).
				Hinted("next_round", baseBallot.Round()).
				Msg("timeout; waiting Proposal; trying to move next round")

			if err := bs.timers.StopTimers([]string{TimerIDBroadcastingINITBallot}); err != nil {
				bs.Log().Error().Err(err).Str("timer", TimerIDBroadcastingINITBallot).Msg("failed to stop")
			}

			ib := baseBallot
			if err := SignSeal(&ib, bs.local); err != nil {
				bs.Log().Error().Err(err).Msg("failed to re-sign ACCEPTBallot, but will keep trying")

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
