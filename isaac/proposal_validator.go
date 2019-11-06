package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hash"
)

type ProposalValidator interface {
	Validated(hash.Hash /* Proposal.Hash() */) bool
	NewBlock(hash.Hash /* Proposal.Hash() */) (Block, error)
}

type BaseProposalValidator struct {
	sealStorage SealStorage
}

func NewBaseProposalValidator(sealStorage SealStorage) BaseProposalValidator {
	return BaseProposalValidator{
		sealStorage: sealStorage,
	}
}

func (bp BaseProposalValidator) GetProposal(proposal hash.Hash) (Proposal, error) {
	if sl, found := bp.sealStorage.Get(proposal); !found {
		return Proposal{}, xerrors.Errorf("failed to find proposal: %v", proposal)
	} else if sl.Type() != ProposalType {
		return Proposal{}, xerrors.Errorf("vr.Proposal() is not proposal: %v", sl.Type())
	} else {
		return sl.(Proposal), nil
	}
}
