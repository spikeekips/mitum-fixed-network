// +build test

package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/hash"
)

type DummyProposalValidator struct {
	sync.RWMutex
	validated map[hash.Hash] /* Proposal.Hash() */ Block
}

func NewDummyProposalValidator() *DummyProposalValidator {
	return &DummyProposalValidator{
		validated: map[hash.Hash]Block{},
	}
}

func (dp *DummyProposalValidator) isValid(proposal Proposal) error {
	if err := proposal.IsValid(); err != nil {
		return err
	}

	// TODO process transactions

	return nil
}

func (dp *DummyProposalValidator) Validated(proposal hash.Hash) bool {
	dp.RLock()
	defer dp.RUnlock()

	_, found := dp.validated[proposal]
	return found
}

func (dp *DummyProposalValidator) NewBlock(height Height, round Round, proposal hash.Hash) (Block, error) {
	dp.Lock()
	defer dp.Unlock()

	if block, found := dp.validated[proposal]; found {
		return block, nil
	}

	// TODO validate proposal
	/*
		if err := dp.isValid(proposal); err != nil {
			return Block{}, err
		}
	*/

	block, err := NewBlock(height, round, proposal)
	if err != nil {
		return Block{}, err
	}

	dp.validated[proposal] = block

	return block, nil
}
