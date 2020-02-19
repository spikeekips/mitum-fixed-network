package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
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
	localState        *LocalState
	proposalProcessor ProposalProcessor
	state             ConsensusState
	stateChan         chan<- ConsensusStateChangeContext
	sealChan          chan<- seal.Seal
}

func NewBaseStateHandler(
	localState *LocalState, proposalProcessor ProposalProcessor, state ConsensusState,
) *BaseStateHandler {
	return &BaseStateHandler{
		localState:        localState,
		proposalProcessor: proposalProcessor,
		state:             state,
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

func (bs *BaseStateHandler) StoreNewBlock(blockStorage BlockStorage) error {
	if err := blockStorage.Commit(); err != nil {
		return err
	}

	_ = bs.localState.SetLastBlock(blockStorage.Block())

	return nil
}

// TODO rename 'vp' to 'voteProof'
func (bs *BaseStateHandler) StoreNewBlockByVoteProof(vp VoteProof) error {
	fact, ok := vp.Majority().(ACCEPTBallotFact)
	if !ok {
		return xerrors.Errorf("needs ACCEPTBallotFact: fact=%T", vp.Majority())
	}

	l := loggerWithVoteProof(vp, bs.Log()).With().
		Str("proposal", fact.Proposal().String()).
		Str("new_block", fact.NewBlock().String()).
		Logger()

	_ = bs.localState.SetLastACCEPTVoteProof(vp)

	l.Debug().Msg("trying to store new block")

	blockStorage, err := bs.proposalProcessor.Process(fact.Proposal(), nil)
	if err != nil {
		return err
	}

	if blockStorage.Block() == nil {
		err := xerrors.Errorf("failed to process Proposal; empty Block returned")
		l.Error().Err(err).Send()

		return err
	}

	if !fact.NewBlock().Equal(blockStorage.Block().Hash()) {
		err := xerrors.Errorf(
			"processed new block does not match; fact=%s processed=%s",
			fact.NewBlock(),
			blockStorage.Block().Hash(),
		)
		l.Error().Err(err).Send()

		return err
	}

	if err := bs.StoreNewBlock(blockStorage); err != nil {
		l.Error().Err(err).Msg("failed to store new block")
		return err
	}

	l.Info().Msg("new block stored")

	return nil
}
