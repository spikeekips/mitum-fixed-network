package isaac

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

type ConsensusStateHandler interface {
	State() ConsensusState
	SetStateChan(chan<- ConsensusStateChangeContext)
	SetSealChan(chan<- seal.Seal)
	Activate(ConsensusStateChangeContext) error
	Deactivate(ConsensusStateChangeContext) error
	// NewSeal receives Seal.
	NewSeal(seal.Seal) error
	// NewVoteProof receives the finished VoteProof.
	NewVoteProof(VoteProof) error
}

type ConsensusStateChangeContext struct {
	fromState ConsensusState
	toState   ConsensusState
	voteProof VoteProof
}

func NewConsensusStateChangeContext(from, to ConsensusState, voteProof VoteProof) ConsensusStateChangeContext {
	return ConsensusStateChangeContext{
		fromState: from,
		toState:   to,
		voteProof: voteProof,
	}
}

func (csc ConsensusStateChangeContext) From() ConsensusState {
	return csc.fromState
}

func (csc ConsensusStateChangeContext) To() ConsensusState {
	return csc.toState
}

func (csc ConsensusStateChangeContext) VoteProof() VoteProof {
	return csc.voteProof
}

type BaseStateHandler struct {
	sync.RWMutex
	*logging.Logger
	localState *LocalState
	state      ConsensusState
	stateChan  chan<- ConsensusStateChangeContext
	sealChan   chan<- seal.Seal
}

func NewBaseStateHandler(localState *LocalState, state ConsensusState) *BaseStateHandler {
	return &BaseStateHandler{
		localState: localState,
		state:      state,
	}
}

func (bs *BaseStateHandler) State() ConsensusState {
	return bs.state
}

func (bs *BaseStateHandler) SetStateChan(stateChan chan<- ConsensusStateChangeContext) {
	bs.stateChan = stateChan
}

func (bs *BaseStateHandler) SetSealChan(sealChan chan<- seal.Seal) {
	bs.sealChan = sealChan
}

func (bs *BaseStateHandler) ChangeState(newState ConsensusState, vp VoteProof) error {
	if bs.stateChan == nil {
		return nil
	}

	if err := newState.IsValid(nil); err != nil {
		return err
	} else if newState == bs.state {
		return xerrors.Errorf("can not change state to same state; current=%s new=%s", bs.state, newState)
	}

	go func() {
		bs.stateChan <- NewConsensusStateChangeContext(bs.state, newState, vp)
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

func loggerWithSeal(sl seal.Seal, l *zerolog.Logger) *zerolog.Logger {
	ll := l.With().
		Str("seal_hint", sl.Hint().Verbose()).
		Str("seal_hash", sl.Hash().String()).
		Logger()

	return &ll
}

func loggerWithBallot(ballot Ballot, l *zerolog.Logger) *zerolog.Logger {
	ll := l.With().
		Str("seal_hint", ballot.Hint().Verbose()).
		Str("seal_hash", ballot.Hash().String()).
		Int64("ballot_height", ballot.Height().Int64()).
		Uint64("ballot_round", ballot.Round().Uint64()).
		Str("ballot_stage", ballot.Stage().String()).
		Logger()

	return &ll
}

func loggerWithVoteProof(vp VoteProof, l *zerolog.Logger) *zerolog.Logger {
	if vp == nil {
		return l
	}

	rvp, _ := util.JSONMarshal(vp)

	ll := l.With().
		Int64("voteproof_height", vp.Height().Int64()).
		Uint64("voteproof_round", vp.Round().Uint64()).
		Str("voteproof_stage", vp.Stage().String()).
		RawJSON("voteproof", rvp).
		Logger()

	return &ll
}

func loggerWithLocalState(localState *LocalState, l *zerolog.Logger) *zerolog.Logger {
	lastBlock := localState.LastBlock()
	if lastBlock == nil {
		return l
	}

	ll := l.With().
		Str("last_block_hash", lastBlock.Hash().String()).
		Int64("last_block_height", lastBlock.Height().Int64()).
		Uint64("last_block_round", lastBlock.Round().Uint64()).
		Logger()

	return &ll
}

func loggerWithConsensusStateChangeContext(ctx ConsensusStateChangeContext, l *zerolog.Logger) *zerolog.Logger {
	ll := l.With().
		Str("from_state", ctx.fromState.String()).
		Str("to_state", ctx.toState.String()).
		Logger()

	return loggerWithVoteProof(ctx.voteProof, &ll)
}
