package isaac

import (
	"sort"
	"strings"
	"sync"

	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Statepool struct {
	sync.RWMutex
	lastManifest block.Manifest
	fromStorage  func(string) (state.State, bool, error)
	cached       map[string]state.StateUpdater
	updated      map[string]state.StateUpdater
}

func NewStatepool(st storage.Storage) (*Statepool, error) {
	var lastManifest block.Manifest
	switch m, found, err := st.LastManifest(); {
	case found:
		lastManifest = m
	case err != nil:
		return nil, err
	}

	return &Statepool{
		fromStorage:  st.State,
		lastManifest: lastManifest,
		cached:       map[string]state.StateUpdater{},
		updated:      map[string]state.StateUpdater{},
	}, nil
}

func (sp *Statepool) Get(key string) (state.StateUpdater, bool, error) {
	sp.Lock()
	defer sp.Unlock()

	if s, found := sp.getFromUpdated(key); found {
		return s, true, nil
	}

	if s, found := sp.getFromCached(key); found {
		return s, true, nil
	}

	var found bool
	var value state.Value
	var previousBlock valuehash.Hash
	switch s, fo, err := sp.fromStorage(key); {
	case err != nil:
		return nil, false, err
	case fo:
		value = s.Value()
		previousBlock = s.CurrentBlock()
		found = fo
	}

	var st state.StateUpdater
	if su, err := state.NewStateV0Updater(key, value, previousBlock); err != nil {
		return nil, found, err
	} else {
		st = su
	}

	sp.cached[key] = st

	return st, found, nil
}

func (sp *Statepool) Set(op valuehash.Hash, s ...state.StateUpdater) error {
	sp.Lock()
	defer sp.Unlock()

	if len(s) < 1 {
		return nil
	}

	for i := range s {
		if err := s[i].AddOperation(op); err != nil {
			return err
		}
	}

	for i := range s {
		if _, found := sp.updated[s[i].Key()]; found {
			continue
		}

		sp.updated[s[i].Key()] = s[i]
	}

	return nil
}

func (sp *Statepool) IsUpdated() bool {
	sp.RLock()
	defer sp.RUnlock()

	return len(sp.updated) > 0
}

func (sp *Statepool) Updates() []state.StateUpdater {
	sp.RLock()
	defer sp.RUnlock()

	us := make([]state.StateUpdater, len(sp.updated))

	var i int
	for s := range sp.updated {
		us[i] = sp.updated[s]
		i++
	}

	sort.Slice(us, func(i, j int) bool {
		return strings.Compare(us[i].Key(), us[j].Key()) < 0
	})

	return us
}

func (sp *Statepool) getFromCached(key string) (state.StateUpdater, bool) {
	s, found := sp.cached[key]
	if found {
		return s, s != nil
	}

	return s, found
}

func (sp *Statepool) getFromUpdated(key string) (state.StateUpdater, bool) {
	s, found := sp.updated[key]

	return s, found
}
