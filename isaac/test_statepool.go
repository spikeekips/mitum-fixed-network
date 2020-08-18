// +build test

package isaac

import (
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
)

func NewStatepoolWithBase(st storage.Storage, base map[string]state.State) (*Statepool, error) {
	if sp, err := NewStatepool(st); err != nil {
		return nil, err
	} else {
		sp.fromStorage = func(key string) (state.State, bool, error) {
			s, found := base[key]
			return s, found, nil
		}

		return sp, nil
	}
}
