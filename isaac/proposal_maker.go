package isaac

import "github.com/spikeekips/mitum/valuehash"

type ProposalMaker struct {
	localstate *Localstate
}

func NewProposalMaker(localstate *Localstate) *ProposalMaker {
	return &ProposalMaker{localstate: localstate}
}

func (pm *ProposalMaker) seals() []valuehash.Hash {
	return nil
}

func (pm *ProposalMaker) Proposal(round Round, b []byte) (Proposal, error) {
	return NewProposalFromLocalstate(pm.localstate, round, pm.seals(), b)
}
