package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ProposalMaker struct {
	sync.Mutex
	localstate *Localstate
	proposed   ballot.Proposal
}

func NewProposalMaker(localstate *Localstate) *ProposalMaker {
	return &ProposalMaker{localstate: localstate}
}

func (pm *ProposalMaker) facts() ([]valuehash.Hash, []valuehash.Hash, error) {
	mo := map[ /* Operation.Fact().Hash() */ string]struct{}{}

	maxOperations := pm.localstate.Policy().MaxOperationsInProposal()

	var facts, seals, uselessSeals []valuehash.Hash
	if err := pm.localstate.Storage().StagedOperationSeals(
		func(sl operation.Seal) (bool, error) {
			var hasOperations bool
			for _, op := range sl.Operations() {
				fh := op.Fact().Hash()
				if _, found := mo[fh.String()]; found {
					continue
				} else if found, err := pm.localstate.Storage().HasOperationFact(fh); err != nil {
					return false, err
				} else if found {
					continue
				}

				facts = append(facts, fh)
				mo[fh.String()] = struct{}{}
				hasOperations = true

				if uint(len(facts)) == maxOperations {
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

	return facts, seals, nil
}

func (pm *ProposalMaker) Proposal(round base.Round) (ballot.Proposal, error) {
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

	facts, seals, err := pm.facts()
	if err != nil {
		return nil, err
	}

	pr := ballot.NewProposalV0(
		pm.localstate.Node().Address(),
		height,
		round,
		facts,
		seals,
	)
	if err := SignSeal(&pr, pm.localstate); err != nil {
		return nil, err
	}

	pm.proposed = pr

	return pr, nil
}
