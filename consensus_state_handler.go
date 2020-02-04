package mitum

import (
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"golang.org/x/xerrors"
)

type ConsensusStateHandler interface {
	State() ConsensusState
	SetStateChan(chan<- ConsensusStateChangeContext)
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

func (bs *BaseStateHandler) ChangeState(newState ConsensusState, vp VoteProof) error {
	if newState == bs.state {
		return xerrors.Errorf("can not change state to same joining state")
	}

	go func() {
		bs.stateChan <- ConsensusStateChangeContext{
			fromState: bs.state,
			toState:   newState,
			voteProof: vp,
		}
	}()

	return nil
}

func (bs *BaseStateHandler) BroadcastSeal(sl seal.Seal, errChan chan<- error) {
	bs.localState.Nodes().Traverse(func(n Node) bool {
		go func(n Node) {
			if err := n.Channel().SendSeal(sl); err != nil {
				if errChan != nil {
					errChan <- err
				}
			}
		}(n)

		return true
	})
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
	ll := l.With().
		Int64("voteproof_height", vp.Height().Int64()).
		Uint64("voteproof_round", vp.Round().Uint64()).
		Str("voteproof_stage", vp.Stage().String()).
		Logger()

	return &ll
}

func loggerWithLocalState(localState *LocalState, l *zerolog.Logger) *zerolog.Logger {
	ll := l.With().
		Str("last_block_hash", localState.LastBlockHash().String()).
		Int64("last_block_height", localState.LastBlockHeight().Int64()).
		Uint64("last_block_round", localState.LastBlockRound().Uint64()).
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
