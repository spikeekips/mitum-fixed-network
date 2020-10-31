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
	local    *Local
	proposed ballot.Proposal
}

func NewProposalMaker(local *Local) *ProposalMaker {
	return &ProposalMaker{local: local}
}

func (pm *ProposalMaker) seals() ([]valuehash.Hash, error) {
	founds := map[ /* Operation.Fact().Hash() */ string]struct{}{}

	maxOperations := pm.local.Policy().MaxOperationsInProposal()

	var facts int
	var seals, uselessSeals []valuehash.Hash
	if err := pm.local.Storage().StagedOperationSeals(
		func(sl operation.Seal) (bool, error) {
			var ofs []valuehash.Hash
			for _, op := range sl.Operations() {
				fh := op.Fact().Hash()
				if _, found := founds[fh.String()]; found {
					continue
				} else if found, err := pm.local.Storage().HasOperationFact(fh); err != nil {
					return false, err
				} else if found {
					continue
				}

				ofs = append(ofs, fh)
				if uint(facts+len(ofs)) > maxOperations {
					break
				}

				founds[fh.String()] = struct{}{}
			}

			switch {
			case uint(facts+len(ofs)) > maxOperations:
				return false, nil
			case len(ofs) > 0:
				facts += len(ofs)
				seals = append(seals, sl.Hash())
			default:
				uselessSeals = append(uselessSeals, sl.Hash())
			}

			return true, nil
		},
		true,
	); err != nil {
		return nil, err
	}

	if len(uselessSeals) > 0 {
		if err := pm.local.Storage().UnstagedOperationSeals(uselessSeals); err != nil {
			return nil, err
		}
	}

	return seals, nil
}

func (pm *ProposalMaker) Proposal(round base.Round) (ballot.Proposal, error) {
	pm.Lock()
	defer pm.Unlock()

	var height base.Height
	switch m, found, err := pm.local.Storage().LastManifest(); {
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

	seals, err := pm.seals()
	if err != nil {
		return nil, err
	}

	pr := ballot.NewProposalV0(
		pm.local.Node().Address(),
		height,
		round,
		seals,
	)
	if err := SignSeal(&pr, pm.local); err != nil {
		return nil, err
	}

	pm.proposed = pr

	return pr, nil
}
