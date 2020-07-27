// +build test

package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/valuehash"
)

type DummyProposalMaker struct {
	sync.Mutex
	localstate *Localstate
	proposed   ballot.Proposal
	sls        []seal.Seal
}

func NewDummyProposalMaker(localstate *Localstate, sls []seal.Seal) *DummyProposalMaker {
	return &DummyProposalMaker{
		localstate: localstate,
		sls:        sls,
	}
}

func (pm *DummyProposalMaker) operations() ([]valuehash.Hash, []valuehash.Hash, error) {
	mo := map[ /* Operation.Hash() */ string]struct{}{}

	maxOperations := pm.localstate.Policy().MaxOperationsInProposal()

	var facts, seals []valuehash.Hash
	for _, sl := range pm.sls {
		var hasOperations bool
		var osl operation.Seal
		if s, ok := sl.(operation.Seal); !ok {
			continue
		} else {
			osl = s
		}

		for _, op := range osl.Operations() {
			if _, found := mo[op.Fact().Hash().String()]; found {
				continue
			} else if found {
				continue
			}

			facts = append(facts, op.Fact().Hash())
			mo[op.Fact().Hash().String()] = struct{}{}
			hasOperations = true

			if uint(len(facts)) == maxOperations {
				break
			}
		}

		if hasOperations {
			seals = append(seals, sl.Hash())
		}
	}

	return facts, seals, nil
}

func (pm *DummyProposalMaker) Proposal(round base.Round) (ballot.Proposal, error) {
	pm.Lock()
	defer pm.Unlock()

	var height base.Height
	switch m, found, err := pm.localstate.Storage().LastManifest(); {
	case !found:
		return nil, storage.NotFoundError.Errorf("last manifest not found")
	case err != nil:
		return nil, err
	default:
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
