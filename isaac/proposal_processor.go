package isaac

import (
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/valuehash"
)

type ProposalProcessor interface {
	ProcessINIT(valuehash.Hash /* Proposal.Hash() */, Voteproof /* INIT Voteproof */, []byte) (Block, error)
	ProcessACCEPT(valuehash.Hash /* Proposal.Hash() */, Voteproof /* ACCEPT Voteproof */, []byte) (BlockStorage, error)
}

type ProposalProcessorV0 struct {
	*logging.Logger
	localstate  *Localstate
	sealStorage SealStorage
	blocks      *sync.Map
}

func NewProposalProcessorV0(localstate *Localstate, sealStorage SealStorage) *ProposalProcessorV0 {
	return &ProposalProcessorV0{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "proposal-processor-v0")
		}),
		localstate:  localstate,
		sealStorage: sealStorage,
		blocks:      &sync.Map{},
	}
}

func (dp *ProposalProcessorV0) ProcessINIT(ph valuehash.Hash, initVoteproof Voteproof, b []byte) (Block, error) {
	if i, found := dp.blocks.Load(ph); found {
		return i.(Block), nil
	}

	var proposal Proposal
	if sl, found, err := dp.sealStorage.Seal(ph); err != nil || !found {
		if err != nil {
			return nil, err
		}

		return nil, xerrors.Errorf("Proposal not found; proposal=%s", ph.String())
	} else { // check proposed time
		ivp := dp.localstate.LastINITVoteproof()
		if ivp == nil {
			return nil, xerrors.Errorf("last INIT Voteproof is missing")
		}

		timespan := dp.localstate.Policy().TimespanValidBallot()
		if sl.SignedAt().Before(ivp.FinishedAt().Add(timespan * -1)) {
			return nil, xerrors.Errorf(
				"Proposal was sent before Voteproof; SignedAt=%s now=%s timespan=%s",
				sl.SignedAt(), ivp.FinishedAt(), timespan,
			)
		}

		proposal = sl.(Proposal)
	}

	lastBlock := dp.localstate.LastBlock()
	if lastBlock == nil {
		return nil, xerrors.Errorf("last block is empty")
	}

	blockOperations, err := dp.processSeals(proposal, b)
	if err != nil {
		return nil, err
	}

	blockStates, err := dp.processStates(proposal, b)
	if err != nil {
		return nil, err
	}

	var block Block
	if b, err := NewBlockV0(
		proposal.Height(), proposal.Round(), proposal.Hash(), lastBlock.Hash(),
		blockOperations, blockStates,
		b,
	); err != nil {
		return nil, err
	} else {
		block = b.SetINITVoteproof(initVoteproof)
	}

	dp.blocks.Store(ph, block)

	return block, nil
}

// TODO b is NetworkID
func (dp *ProposalProcessorV0) ProcessACCEPT(
	ph valuehash.Hash, acceptVoteproof Voteproof, _ []byte,
) (BlockStorage, error) {
	var block Block
	if i, found := dp.blocks.Load(ph); !found {
		return nil, xerrors.Errorf("not processed ProcessINIT")
	} else {
		block = i.(Block).SetACCEPTVoteproof(acceptVoteproof)
	}

	return dp.localstate.Storage().OpenBlockStorage(block)
}

func (dp *ProposalProcessorV0) processSeals(Proposal, []byte) (valuehash.Hash, error) {
	return nil, nil
}

func (dp *ProposalProcessorV0) processStates(Proposal, []byte) (valuehash.Hash, error) {
	return nil, nil
}
