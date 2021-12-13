package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ProposalMaker struct {
	sync.Mutex
	local    *node.Local
	database storage.Database
	policy   *LocalPolicy
	proposed base.Proposal
}

func NewProposalMaker(
	local *node.Local,
	db storage.Database,
	policy *LocalPolicy,
) *ProposalMaker {
	return &ProposalMaker{local: local, database: db, policy: policy}
}

func (pm *ProposalMaker) operations() ([]valuehash.Hash, error) {
	founds := map[ /* Operation.Fact().Hash() */ string]struct{}{}

	maxOperations := pm.policy.MaxOperationsInProposal()

	var ops, uselesses []valuehash.Hash
	if err := pm.database.StagedOperations(
		func(op operation.Operation) (bool, error) {
			fh := op.Fact().Hash()
			if _, found := founds[fh.String()]; found {
				uselesses = append(uselesses, fh)

				return true, nil
			}

			switch found, err := pm.database.HasOperationFact(fh); {
			case err != nil:
				return false, err
			case found:
				uselesses = append(uselesses, fh)

				return true, nil
			}

			ops = append(ops, fh)
			if uint(len(ops)) == maxOperations {
				return false, nil
			}

			founds[fh.String()] = struct{}{}

			return true, nil
		},
		true,
	); err != nil {
		return nil, err
	}

	if len(uselesses) > 0 {
		if err := pm.database.UnstagedOperations(uselesses); err != nil {
			return nil, err
		}
	}

	return ops, nil
}

func (pm *ProposalMaker) Proposal(
	height base.Height,
	round base.Round,
	voteproof base.Voteproof,
) (base.Proposal, error) {
	pm.Lock()
	defer pm.Unlock()

	if pm.proposed != nil {
		if pm.proposed.Fact().Height() == height && pm.proposed.Fact().Round() == round {
			return pm.proposed, nil
		}
	}

	ops, err := pm.operations()
	if err != nil {
		return nil, err
	}

	pr, err := ballot.NewProposal(
		ballot.NewProposalFact(
			height,
			round,
			pm.local.Address(),
			ops,
		),
		pm.local.Address(),
		voteproof,
		pm.local.Privatekey(), pm.policy.NetworkID(),
	)
	if err != nil {
		return nil, err
	}

	pm.proposed = pr

	return pr, nil
}
