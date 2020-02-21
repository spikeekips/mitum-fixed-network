package isaac

import (
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
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

func (bs *BaseStateHandler) ChangeState(newState State, vp Voteproof) error {
	if bs.stateChan == nil {
		return nil
	}

	if err := newState.IsValid(nil); err != nil {
		return err
	} else if newState == bs.state {
		return xerrors.Errorf("can not change state to same state; current=%s new=%s", bs.state, newState)
	}

	go func() {
		bs.stateChan <- NewStateChangeContext(bs.state, newState, vp)
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

// TODO rename 'vp' to 'voteproof'
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

	blockStorage, err := bs.proposalProcessor.ProcessACCEPT(fact.Proposal(), acceptVoteproof, nil)
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

	l.Info().Dict("block", zerolog.Dict().
		Str("proposal", blockStorage.Block().Proposal().String()).
		Str("hash", blockStorage.Block().Hash().String()).
		Int64("height", blockStorage.Block().Height().Int64()).
		Uint64("round", blockStorage.Block().Round().Uint64()),
	).
		Msg("new block stored")

	return nil
}
