package isaac

import "github.com/spikeekips/mitum/valuehash"

type ProposalMaker struct {
	localState *LocalState
}

func NewProposalMaker(localState *LocalState) *ProposalMaker {
	return &ProposalMaker{localState: localState}
}

func (pm ProposalMaker) seals() []valuehash.Hash {
	return nil
}

func (pm ProposalMaker) Proposal(round Round, b []byte) (Proposal, error) {
	return NewProposalFromLocalState(pm.localState, round, pm.seals(), b)
}
