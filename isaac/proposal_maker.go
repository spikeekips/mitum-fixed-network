package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/valuehash"
)

type ProposalMaker struct {
	sync.Mutex
	localstate *Localstate
	proposed   Proposal
}

func NewProposalMaker(localstate *Localstate) *ProposalMaker {
	return &ProposalMaker{localstate: localstate}
}

func (pm *ProposalMaker) seals() ([]valuehash.Hash, error) {
	// TODO to reduce the marshal/unmarshal, consider to get the hashes for
	// staged like 'StagedOperationSealHashes'.
	var seals []valuehash.Hash
	if err := pm.localstate.Storage().StagedOperationSeals(
		func(sl operation.Seal) (bool, error) {
			// TODO check the duplication of Operation.Hash
			seals = append(seals, sl.Hash())

			return len(seals) != operation.MaxOperationsInSeal, nil
		},
		true,
	); err != nil {
		return nil, err
	}

	return seals, nil
}

func (pm *ProposalMaker) Proposal(round Round) (Proposal, error) {
	pm.Lock()
	defer pm.Unlock()

	lastBlock := pm.localstate.LastBlock()

	height := lastBlock.Height() + 1

	if pm.proposed != nil {
		if pm.proposed.Height() == height && pm.proposed.Round() == round {
			return pm.proposed, nil
		}
	}

	seals, err := pm.seals()
	if err != nil {
		return nil, err
	}

	proposal, err := NewProposal(pm.localstate, height, round, seals, pm.localstate.Policy().NetworkID())
	if err != nil {
		return nil, err
	}

	pm.proposed = proposal

	return proposal, nil
}
