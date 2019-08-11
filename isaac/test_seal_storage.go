package isaac

import (
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/seal"
	"golang.org/x/sync/syncmap"
	"golang.org/x/xerrors"
)

type TestSealStorage struct {
	m *syncmap.Map
}

func NewTestSealStorage() *TestSealStorage {
	return &TestSealStorage{
		m: &syncmap.Map{},
	}
}

func (tss *TestSealStorage) Has(h hash.Hash) bool {
	_, found := tss.m.Load(h)
	return found
}

func (tss *TestSealStorage) Get(h hash.Hash) seal.Seal {
	if s, found := tss.m.Load(h); !found {
		return nil
	} else if sl, ok := s.(seal.Seal); !ok {
		return nil
	} else {
		return sl
	}
}

func (tss *TestSealStorage) Save(sl seal.Seal) error {
	if tss.Has(sl.Hash()) {
		return xerrors.Errorf("already stored; %v", sl.Hash())
	}

	tss.m.Store(sl.Hash(), sl)

	return nil
}
