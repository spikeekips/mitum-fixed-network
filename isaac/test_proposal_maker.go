//go:build test
// +build test

package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/valuehash"
)

type DummyProposalMaker struct {
	sync.Mutex
	local    *Local
	proposed base.Proposal
	sls      []seal.Seal
}

func NewDummyProposalMaker(local *Local, sls []seal.Seal) *DummyProposalMaker {
	return &DummyProposalMaker{
		local: local,
		sls:   sls,
	}
}

func (pm *DummyProposalMaker) seals() ([]valuehash.Hash, error) {
	mo := map[ /* Operation.Hash() */ string]struct{}{}

	maxOperations := pm.local.Policy().MaxOperationsInProposal()

	var facts int
	var seals []valuehash.Hash
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

			facts++
			mo[op.Fact().Hash().String()] = struct{}{}
			hasOperations = true

			if uint(facts) == maxOperations {
				break
			}
		}

		if hasOperations {
			seals = append(seals, sl.Hash())
		}
	}

	return seals, nil
}

func (pm *DummyProposalMaker) Proposal(height base.Height, round base.Round, voteproof base.Voteproof) (base.Proposal, error) {
	pm.Lock()
	defer pm.Unlock()

	if pm.proposed != nil {
		if pm.proposed.Fact().Height() == height && pm.proposed.Fact().Round() == round {
			return pm.proposed, nil
		}
	}

	seals, err := pm.seals()
	if err != nil {
		return nil, err
	}

	pr, err := ballot.NewProposal(
		ballot.NewProposalFact(
			height,
			round,
			pm.local.Node().Address(),
			seals,
		),
		pm.local.Node().Address(),
		voteproof,
		pm.local.Node().Privatekey(), pm.local.Policy().NetworkID(),
	)
	if err != nil {
		return nil, err
	}
	pm.proposed = pr

	return pr, nil
}
