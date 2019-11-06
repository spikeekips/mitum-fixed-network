// +build test

package isaac

import (
	"fmt"

	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
	"golang.org/x/sync/syncmap"
	"golang.org/x/xerrors"
)

type TSealStorage struct {
	m         *syncmap.Map
	proposals *syncmap.Map
}

func NewTSealStorage() *TSealStorage {
	return &TSealStorage{
		m:         &syncmap.Map{},
		proposals: &syncmap.Map{},
	}
}

func (tss *TSealStorage) Has(h hash.Hash) bool {
	_, found := tss.m.Load(h)
	return found
}

func (tss *TSealStorage) Get(h hash.Hash) (seal.Seal, bool) {
	if s, found := tss.m.Load(h); !found {
		return nil, false
	} else if sl, ok := s.(seal.Seal); !ok {
		return nil, false
	} else {
		return sl, true
	}
}

func (tss *TSealStorage) GetProposal(n node.Address, height Height, round Round) (Proposal, bool) {
	k := tss.proposalKey(n, height, round)
	if s, found := tss.proposals.Load(k); !found {
		return Proposal{}, false
	} else if h, ok := s.(hash.Hash); !ok {
		return Proposal{}, false
	} else if i, ok := tss.Get(h); !ok {
		return Proposal{}, false
	} else if sl, ok := i.(Proposal); !ok {
		return Proposal{}, false
	} else {
		return sl, true
	}
}

func (tss *TSealStorage) proposalKey(n node.Address, height Height, round Round) string {
	return fmt.Sprintf("%s-%s-%d", n.String(), height.String(), round)
}

func (tss *TSealStorage) Save(sl seal.Seal) error {
	if sl == nil {
		return xerrors.Errorf("seal should not be nil")
	}

	if tss.Has(sl.Hash()) {
		return xerrors.Errorf("already stored; %v", sl.Hash())
	}

	tss.m.Store(sl.Hash(), sl)

	switch sl.Type() {
	case ProposalType:
		proposal, ok := sl.(Proposal)
		if !ok {
			return xerrors.Errorf("seal.Type() is proposal, but it's not; message=%q", sl)
		}

		tss.proposals.Store(
			tss.proposalKey(proposal.Proposer(), proposal.Height(), proposal.Round()),
			sl.Hash(),
		)
	}

	return nil
}
