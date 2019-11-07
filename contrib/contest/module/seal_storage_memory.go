package contest_module

import (
	"fmt"

	"github.com/rs/zerolog"
	"golang.org/x/sync/syncmap"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type MemorySealStorage struct {
	*common.Logger
	m         *syncmap.Map
	proposals *syncmap.Map
}

func NewMemorySealStorage() *MemorySealStorage {
	return &MemorySealStorage{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "memory-seal-storage")
		}),
		m:         &syncmap.Map{},
		proposals: &syncmap.Map{},
	}
}

func (mss *MemorySealStorage) Has(h hash.Hash) bool {
	_, found := mss.m.Load(h)
	return found
}

func (mss *MemorySealStorage) Get(h hash.Hash) (seal.Seal, bool) {
	if i, found := mss.m.Load(h); !found {
		return nil, false
	} else if sl, ok := i.(seal.Seal); !ok {
		return nil, false
	} else {
		return sl, true
	}
}

func (mss *MemorySealStorage) GetProposal(n node.Address, height isaac.Height, round isaac.Round) (isaac.Proposal, bool) {
	k := mss.proposalKey(n, height, round)
	if s, found := mss.proposals.Load(k); !found {
		return isaac.Proposal{}, false
	} else if h, ok := s.(hash.Hash); !ok {
		return isaac.Proposal{}, false
	} else if i, ok := mss.Get(h); !ok {
		return isaac.Proposal{}, false
	} else if sl, ok := i.(isaac.Proposal); !ok {
		return isaac.Proposal{}, false
	} else {
		return sl, true
	}
}

func (mss *MemorySealStorage) proposalKey(n node.Address, height isaac.Height, round isaac.Round) string {
	return fmt.Sprintf("%s-%s-%d", n.String(), height.String(), round)
}

func (mss *MemorySealStorage) Save(sl seal.Seal) error {
	if sl == nil {
		return xerrors.Errorf("seal should not be nil")
	}

	if mss.Has(sl.Hash()) {
		return xerrors.Errorf("already stored; %v", sl.Hash())
	}

	mss.m.Store(sl.Hash(), sl)

	switch sl.Type() {
	case isaac.ProposalType:
		proposal, ok := sl.(isaac.Proposal)
		if !ok {
			return xerrors.Errorf("seal.Type() is proposal, but it's not; message=%q", sl)
		}

		mss.proposals.Store(
			mss.proposalKey(proposal.Proposer(), proposal.Height(), proposal.Round()),
			sl.Hash(),
		)
	}

	//mss.Log().Debug().Object("seal", sl.Hash()).Msg("seal saved")
	return nil
}
