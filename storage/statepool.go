package storage

import (
	"sort"
	"strings"
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type cachedState struct {
	state.State
	exists bool
}

func newCachedState(st state.State, exists bool) cachedState {
	return cachedState{State: st, exists: exists}
}

type Statepool struct {
	sync.RWMutex
	nextHeight  base.Height
	fromStorage func(string) (state.State, bool, error)
	cached      map[string]cachedState
	updated     map[string]*state.StateUpdater
	insertedOps map[string]valuehash.Hash
	addedOps    map[string]operation.Operation
}

func NewStatepool(st Database) (*Statepool, error) {
	nextHeight := base.GenesisHeight
	switch m, found, err := st.LastManifest(); {
	case err != nil:
		return nil, err
	case found:
		nextHeight = m.Height() + 1
	}

	return &Statepool{
		fromStorage: st.State,
		nextHeight:  nextHeight,
		cached:      map[string]cachedState{},
		updated:     map[string]*state.StateUpdater{},
		insertedOps: map[string]valuehash.Hash{},
		addedOps:    map[string]operation.Operation{},
	}, nil
}

// NewStatepoolWithBase only used for testing
func NewStatepoolWithBase(st Database, b map[string]state.State) (*Statepool, error) {
	sp, err := NewStatepool(st)
	if err != nil {
		return nil, err
	}
	sp.fromStorage = func(key string) (state.State, bool, error) {
		if s, found := b[key]; found {
			return s, true, nil
		}
		return st.State(key)
	}

	return sp, nil
}

func (sp *Statepool) Get(key string) (state.State, bool, error) {
	sp.Lock()
	defer sp.Unlock()

	st, exists, err := sp.get(key)
	if err != nil {
		return nil, false, err
	}
	return st.Clear(), exists, nil
}

func (sp *Statepool) get(key string) (state.State, bool, error) {
	if ca, found := sp.cached[key]; found {
		return ca.State, ca.exists, nil
	}

	switch st, found, err := sp.fromStorage(key); {
	case err != nil:
		return nil, false, err
	case found:
		st = st.Clear()
		sp.cached[key] = newCachedState(st, true)

		return st, true, nil
	}

	st, err := state.NewStateV0(key, nil, base.NilHeight)
	if err != nil {
		return nil, false, err
	}
	sp.cached[key] = newCachedState(st, false)

	return st, false, nil
}

func (sp *Statepool) Set(fact valuehash.Hash, s ...state.State) error {
	if len(s) < 1 {
		return nil
	}

	sp.Lock()
	defer sp.Unlock()

	for i := range s {
		st := s[i]

		var su *state.StateUpdater
		if u, found := sp.updated[s[i].Key()]; found {
			su = u
		} else {
			nu := state.NewStateUpdater(st.Clear()).SetHeight(sp.nextHeight)

			sp.updated[s[i].Key()] = nu
			su = nu
		}

		if err := func() error {
			if err := su.Merge(st); err != nil {
				return err
			}

			return su.AddOperation(fact)
		}(); err != nil {
			err0 := util.NewError("failed to set States").Merge(err)
			for j := 0; j <= i; j++ {
				// NOTE reset previous updated states
				if err := sp.updated[s[j].Key()].Reset(); err != nil {
					return err0.Wrap(err)
				}
			}

			return err0
		}
	}

	sp.insertOperations(fact)

	return nil
}

func (sp *Statepool) IsUpdated() bool {
	sp.RLock()
	defer sp.RUnlock()

	return len(sp.updated) > 0
}

func (sp *Statepool) Updates() []*state.StateUpdater {
	sp.RLock()
	defer sp.RUnlock()

	us := make([]*state.StateUpdater, len(sp.updated))

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

func (sp *Statepool) insertOperations(facts ...valuehash.Hash) {
	for i := range facts {
		f := facts[i]
		if _, found := sp.insertedOps[f.String()]; !found {
			sp.insertedOps[f.String()] = f
		}
	}
}

func (sp *Statepool) InsertedOperations() map[string]valuehash.Hash {
	sp.RLock()
	defer sp.RUnlock()

	return sp.insertedOps
}

func (sp *Statepool) Height() base.Height {
	return sp.nextHeight
}

func (sp *Statepool) AddOperations(ops ...operation.Operation) {
	sp.RLock()
	defer sp.RUnlock()

	for i := range ops {
		op := ops[i]
		f := op.Fact().Hash()
		if _, found := sp.addedOps[f.String()]; !found {
			sp.addedOps[f.String()] = op
		}

		sp.insertOperations(f)
	}
}

func (sp *Statepool) AddedOperations() map[string]operation.Operation {
	sp.RLock()
	defer sp.RUnlock()

	return sp.addedOps
}

// Done cleans up the resources.
func (sp *Statepool) Done() {
	sp.Lock()
	defer sp.Unlock()

	for s := range sp.updated {
		state.StateUpdaterPoolPut(sp.updated[s])
	}

	sp.cached = nil
	sp.updated = nil
	sp.insertedOps = nil
	sp.addedOps = nil
}
