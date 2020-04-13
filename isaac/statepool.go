package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
)

type StatePool struct {
	sync.RWMutex
	st      storage.Storage
	cached  map[string]state.StateUpdater
	updated map[string]state.StateUpdater
}

func NewStatePool(st storage.Storage) *StatePool {
	return &StatePool{
		st:      st,
		cached:  map[string]state.StateUpdater{},
		updated: map[string]state.StateUpdater{},
	}
}

func (sp *StatePool) Get(key string) (state.StateUpdater, error) {
	if s, found := sp.getFromUpdated(key); found {
		return s, nil
	}

	if s, found := sp.getFromCached(key); found {
		return s, nil
	}

	var value state.Value
	var previousBlock valuehash.Hash

	if s, _, err := sp.st.State(key); err != nil {
		return nil, err
	} else if s != nil {
		value = s.Value()
		previousBlock = s.CurrentBlock()
	}

	var st state.StateUpdater
	if su, err := state.NewStateV0(key, value, previousBlock); err != nil {
		return nil, err
	} else {
		st = su
	}

	sp.Lock()
	defer sp.Unlock()
	sp.cached[key] = st

	return st, nil
}

func (sp *StatePool) Set(s state.StateUpdater) error {
	sp.Lock()
	defer sp.Unlock()

	sp.updated[s.Key()] = s

	return nil
}

func (sp *StatePool) Updated() []state.StateUpdater {
	us := make([]state.StateUpdater, len(sp.updated))

	var i int
	for _, s := range sp.updated {
		us[i] = s
		i++
	}

	return us
}

func (sp *StatePool) getFromCached(key string) (state.StateUpdater, bool) {
	sp.RLock()
	defer sp.RUnlock()

	s, found := sp.cached[key]
	if found {
		return s, s != nil
	}

	return s, found
}

func (sp *StatePool) getFromUpdated(key string) (state.StateUpdater, bool) {
	sp.RLock()
	defer sp.RUnlock()

	s, found := sp.updated[key]

	return s, found
}
