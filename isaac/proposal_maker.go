package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/valuehash"
)

type ProposalMaker struct {
	sync.Mutex
	localstate *Localstate
	proposed   Proposal
}

func NewProposalMaker(localstate *Localstate) *ProposalMaker {
	return &ProposalMaker{localstate: localstate}
}

func (pm *ProposalMaker) seals() []valuehash.Hash {
	return nil
}

func (pm *ProposalMaker) Proposal(round Round, b []byte) (Proposal, error) {
	pm.Lock()
	defer pm.Unlock()

	lastBlock := pm.localstate.LastBlock()

	height := lastBlock.Height() + 1

	if pm.proposed != nil {
		if pm.proposed.Height() == height && pm.proposed.Round() == round {
			return pm.proposed, nil
		}
	}

	proposal, err := NewProposal(pm.localstate, height, round, pm.seals(), b)
	if err != nil {
		return nil, err
	}

	pm.proposed = proposal

	return proposal, nil
}
