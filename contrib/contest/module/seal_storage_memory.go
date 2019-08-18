package contest_module

import (
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/seal"
	"golang.org/x/sync/syncmap"
	"golang.org/x/xerrors"
)

type MemorySealStorage struct {
	*common.Logger
	m *syncmap.Map
}

func NewMemorySealStorage() *MemorySealStorage {
	return &MemorySealStorage{
		Logger: common.NewLogger(log, "module", "memory-seal-storage"),
		m:      &syncmap.Map{},
	}
}

func (tss *MemorySealStorage) Has(h hash.Hash) bool {
	_, found := tss.m.Load(h)
	return found
}

func (tss *MemorySealStorage) Get(h hash.Hash) seal.Seal {
	if s, found := tss.m.Load(h); !found {
		return nil
	} else if sl, ok := s.(seal.Seal); !ok {
		return nil
	} else {
		return sl
	}
}

func (tss *MemorySealStorage) Save(sl seal.Seal) error {
	if sl == nil {
		return xerrors.Errorf("seal should not be nil")
	}

	if tss.Has(sl.Hash()) {
		return xerrors.Errorf("already stored; %v", sl.Hash())
	}

	tss.m.Store(sl.Hash(), sl)

	//tss.Log().Debug("seal saved", "seal", sl.Hash())
	return nil
}
