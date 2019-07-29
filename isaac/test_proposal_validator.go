// +build test

package isaac

type DummyProposalValidator struct {
}

func NewDummyProposalValidator() *DummyProposalValidator {
	return &DummyProposalValidator{}
}

func (dp *DummyProposalValidator) isValid(proposal Proposal) error {
	if err := proposal.IsValid(); err != nil {
		return err
	}

	// TODO process transactions

	return nil
}

func (dp *DummyProposalValidator) NewBlock(proposal Proposal) (Block, error) {
	if err := dp.isValid(proposal); err != nil {
		return Block{}, err
	}

	return NewBlock(proposal.Height(), proposal.Round(), proposal.Hash())
}
