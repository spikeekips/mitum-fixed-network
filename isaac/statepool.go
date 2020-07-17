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
	st           storage.Storage
	lastManifest block.Manifest
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
		st:           st,
		lastManifest: lastManifest,
		cached:       map[string]state.StateUpdater{},
		updated:      map[string]state.StateUpdater{},
	}, nil
}

func (sp *Statepool) Get(key string) (state.StateUpdater, bool, error) {
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

func (sp *Statepool) Set(s ...state.StateUpdater) error {
	sp.Lock()
	defer sp.Unlock()

	for i := range s {
		sp.updated[s[i].Key()] = s[i]
	}

	return nil
}

func (sp *Statepool) IsUpdated() bool {
	return len(sp.updated) > 0
}

func (sp *Statepool) Updates() []state.StateUpdater {
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
	sp.RLock()
	defer sp.RUnlock()

	s, found := sp.cached[key]
	if found {
		return s, s != nil
	}

	return s, found
}

func (sp *Statepool) getFromUpdated(key string) (state.StateUpdater, bool) {
	sp.RLock()
	defer sp.RUnlock()

	s, found := sp.updated[key]

	return s, found
}
