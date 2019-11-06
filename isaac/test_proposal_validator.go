// +build test

package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/hash"
)

type DummyProposalValidator struct {
	sync.RWMutex
	BaseProposalValidator
	sealStorage SealStorage
	validated   map[hash.Hash] /* Proposal.Hash() */ Block
}

func NewDummyProposalValidator(sealStorage SealStorage) *DummyProposalValidator {
	return &DummyProposalValidator{
		BaseProposalValidator: NewBaseProposalValidator(sealStorage),
		validated:             map[hash.Hash]Block{},
		sealStorage:           sealStorage,
	}
}

func (dp *DummyProposalValidator) Validated(proposal hash.Hash) bool {
	dp.RLock()
	defer dp.RUnlock()

	_, found := dp.validated[proposal]
	return found
}

func (dp *DummyProposalValidator) NewBlock(h hash.Hash) (Block, error) {
	dp.Lock()
	defer dp.Unlock()

	if block, found := dp.validated[h]; found {
		return block, nil
	}

	proposal, err := dp.BaseProposalValidator.GetProposal(h)
	if err != nil {
		return Block{}, err
	}

	block, err := NewBlock(proposal.Height(), proposal.Round(), proposal.Hash())
	if err != nil {
		return Block{}, err
	}

	dp.validated[proposal.Hash()] = block

	return block, nil
}
