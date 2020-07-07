package isaac

import (
	"sync"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/valuehash"
)

type StatePool struct {
	sync.RWMutex
	st           storage.Storage
	lastManifest block.Manifest
	cached       map[string]state.StateUpdater
	updated      map[string]state.StateUpdater
}

func NewStatePool(st storage.Storage) (*StatePool, error) {
	var lastManifest block.Manifest
	switch m, found, err := st.LastManifest(); {
	case found:
		lastManifest = m
	case err != nil:
		return nil, err
	}

	return &StatePool{
		st:           st,
		lastManifest: lastManifest,
		cached:       map[string]state.StateUpdater{},
		updated:      map[string]state.StateUpdater{},
	}, nil
}

func (sp *StatePool) Get(key string) (state.StateUpdater, bool, error) {
	if s, found := sp.getFromUpdated(key); found {
		return s, true, nil
	}

	if s, found := sp.getFromCached(key); found {
		return s, true, nil
	}

	var found bool
	var value state.Value
	var previousBlock valuehash.Hash
	switch s, fo, err := sp.st.State(key); {
	case err != nil:
		return nil, false, err
	case fo:
		value = s.Value()
		previousBlock = s.CurrentBlock()
		found = fo
	case sp.lastManifest != nil:
		previousBlock = sp.lastManifest.Hash()
	}

	var st state.StateUpdater
	if su, err := state.NewStateV0(key, value, previousBlock); err != nil {
		return nil, found, err
	} else {
		st = su
	}

	sp.Lock()
	defer sp.Unlock()

	sp.cached[key] = st

	return st, found, nil
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
