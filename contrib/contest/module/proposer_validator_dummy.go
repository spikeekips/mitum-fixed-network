package contest_module

import (
	"sync"

	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
)

type DummyProposalValidator struct {
	sync.RWMutex
	validated map[hash.Hash] /* Proposal.Hash() */ isaac.Block
}

func NewDummyProposalValidator() *DummyProposalValidator {
	return &DummyProposalValidator{
		validated: map[hash.Hash]isaac.Block{},
	}
}

func (dp *DummyProposalValidator) Validated(proposal hash.Hash) bool {
	dp.RLock()
	defer dp.RUnlock()

	_, found := dp.validated[proposal]
	return found
}

func (dp *DummyProposalValidator) NewBlock(height isaac.Height, round isaac.Round, proposal hash.Hash) (isaac.Block, error) {
	dp.Lock()
	defer dp.Unlock()

	if block, found := dp.validated[proposal]; found {
		return block, nil
	}

	block, err := isaac.NewBlock(height, round, proposal)
	if err != nil {
		return isaac.Block{}, err
	}

	dp.validated[proposal] = block

	return block, nil
}
