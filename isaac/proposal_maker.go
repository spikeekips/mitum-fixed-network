package isaac

import (
	"fmt"
	"sync"

	"github.com/spikeekips/mitum/valuehash"
)

type ProposalMaker struct {
	sync.Mutex
	localstate *Localstate
	proposed   map[string]Proposal
}

func NewProposalMaker(localstate *Localstate) *ProposalMaker {
	return &ProposalMaker{localstate: localstate, proposed: map[string]Proposal{}}
}

func (pm *ProposalMaker) seals() []valuehash.Hash {
	return nil
}

func (pm *ProposalMaker) Proposal(round Round, b []byte) (Proposal, error) {
	pm.Lock()
	defer pm.Unlock()

	lastBlock := pm.localstate.LastBlock()

	height := lastBlock.Height() + 1
	key := fmt.Sprintf("%d-%d", height.Int64(), round)

	if p, found := pm.proposed[key]; found {
		return p, nil
	}

	proposal, err := NewProposal(pm.localstate, height, round, pm.seals(), b)
	if err != nil {
		return nil, err
	}

	pm.proposed[key] = proposal

	return proposal, nil
}

func (pm *ProposalMaker) Clean() {
	pm.Lock()
	defer pm.Unlock()

	lastBlock := pm.localstate.LastBlock()

	var remove []string
	for key := range pm.proposed {
		var height int64
		var round uint64
		if n, err := fmt.Sscanf(key, "%d-%d", &height, &round); err != nil || n != 2 {
			remove = append(remove, key)
		}
		if height <= lastBlock.Height().Int64() {
			remove = append(remove, key)
		}
	}

	for _, key := range remove {
		delete(pm.proposed, key)
	}
}
