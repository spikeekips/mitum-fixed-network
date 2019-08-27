package contest_module

import (
	"sync"

	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
)

type DummyProposalValidator struct {
	validated *sync.Map
}

func NewDummyProposalValidator() *DummyProposalValidator {
	return &DummyProposalValidator{
		validated: &sync.Map{},
	}
}

func (dp *DummyProposalValidator) Validated(proposal hash.Hash) bool {
	_, found := dp.validated.Load(proposal)
	return found
}

func (dp *DummyProposalValidator) NewBlock(height isaac.Height, round isaac.Round, proposal hash.Hash) (isaac.Block, error) {
	if i, found := dp.validated.Load(proposal); found {
		return i.(isaac.Block), nil
	}

	block, err := isaac.NewBlock(height, round, proposal)
	if err != nil {
		return isaac.Block{}, err
	}

	dp.validated.Store(proposal, block)

	return block, nil
}
