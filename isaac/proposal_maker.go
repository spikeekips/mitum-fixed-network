package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/valuehash"
)

type ProposalMaker struct {
	sync.Mutex
	localstate *Localstate
	proposed   ballot.Proposal
}

func NewProposalMaker(localstate *Localstate) *ProposalMaker {
	return &ProposalMaker{localstate: localstate}
}

func (pm *ProposalMaker) operations() ([]valuehash.Hash, []valuehash.Hash, error) {
	// TODO to reduce the marshal/unmarshal, consider to get the hashes for
	// staged like 'StagedOperationSealHashes'.

	mo := map[ /* Operation.Hash() */ valuehash.Hash]struct{}{}

	var operations, seals, uselessSeals []valuehash.Hash
	if err := pm.localstate.Storage().StagedOperationSeals(
		func(sl operation.Seal) (bool, error) {
			var hasOperations bool
			for _, op := range sl.OperationHashes() {
				if _, found := mo[op]; found {
					continue
				} else if found, err := pm.localstate.Storage().HasOperation(op); err != nil {
					return false, err
				} else if found {
					continue
				}

				operations = append(operations, op)
				mo[op] = struct{}{}
				hasOperations = true

				if len(operations) == operation.MaxOperationsInSeal {
					return false, nil
				}
			}

			if hasOperations {
				seals = append(seals, sl.Hash())
			} else {
				uselessSeals = append(uselessSeals, sl.Hash())
			}

			return true, nil
		},
		true,
	); err != nil {
		return nil, nil, err
	}

	if len(uselessSeals) > 0 {
		if err := pm.localstate.Storage().UnstagedOperationSeals(uselessSeals); err != nil {
			return nil, nil, err
		}
	}

	return operations, seals, nil
}

func (pm *ProposalMaker) Proposal(round base.Round) (ballot.Proposal, error) {
	pm.Lock()
	defer pm.Unlock()

	var height base.Height
	if m, err := pm.localstate.Storage().LastManifest(); err != nil {
		return nil, err
	} else {
		height = m.Height() + 1
	}

	if pm.proposed != nil {
		if pm.proposed.Height() == height && pm.proposed.Round() == round {
			return pm.proposed, nil
		}
	}

	operations, seals, err := pm.operations()
	if err != nil {
		return nil, err
	}

	pr := ballot.NewProposalV0(
		pm.localstate.Node().Address(),
		height,
		round,
		operations,
		seals,
	)
	if err := SignSeal(&pr, pm.localstate); err != nil {
		return nil, err
	}

	pm.proposed = pr

	return pr, nil
}
