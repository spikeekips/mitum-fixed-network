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

type cachedState struct {
	state.State
	exists bool
}

type Statepool struct {
	sync.RWMutex
	lastManifest block.Manifest
	fromStorage  func(string) (state.State, bool, error)
	cached       map[string]cachedState
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
		cached:       map[string]cachedState{},
		updated:      map[string]state.StateUpdater{},
	}, nil
}

func (sp *Statepool) Get(key string) (state.State, bool, error) {
	sp.Lock()
	defer sp.Unlock()

	if ca, found := sp.cached[key]; found {
		return ca.State, ca.exists, nil
	}

	switch st, found, err := sp.fromStorage(key); {
	case err != nil:
		return nil, false, err
	case found:
		sp.cached[key] = cachedState{State: st, exists: true}

		return st, true, nil
	}

	if st, err := state.NewStateV0(key, nil, nil); err != nil {
		return nil, false, err
	} else {
		sp.cached[key] = cachedState{State: st, exists: false}

		return st, false, nil
	}
}

func (sp *Statepool) Set(op valuehash.Hash, s ...state.State) error {
	if len(s) < 1 {
		return nil
	}

	sp.Lock()
	defer sp.Unlock()

	for i := range s {
		st := s[i]

		var su state.StateUpdater
		if u, found := sp.updated[s[i].Key()]; !found {
			if nu, err := state.NewStateV0Updater(st.Key(), st.Value(), st.PreviousBlock()); err != nil {
				return err
			} else {
				sp.updated[s[i].Key()] = nu
				su = nu
			}
		} else {
			su = u
		}

		if err := func() error {
			if _, err := su.Merge(st); err != nil {
				return err
			} else if err := su.AddOperation(op); err != nil {
				return err
			}

			return nil
		}(); err != nil {
			for j := 0; j <= i; j++ {
				sp.updated[s[j].Key()].Reset() // NOTE reset previous updated states
			}

			return err
		}
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
