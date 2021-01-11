package process

import (
	"fmt"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/logging"
)

type ActinfSuffrageElectFunc func(base.Height, base.Round) (base.ActingSuffrage, error)

type BaseSuffrage struct {
	sync.RWMutex
	*logging.Logging
	name           string
	local          *isaac.Local
	numberOfActing uint
	cacheSize      int
	cache          *lru.TwoQueueCache
	electFunc      ActinfSuffrageElectFunc
}

func NewBaseSuffrage(
	name string,
	local *isaac.Local,
	cacheSize int,
	numberOfActing uint,
	electFunc ActinfSuffrageElectFunc,
) *BaseSuffrage {
	var cache *lru.TwoQueueCache
	if cacheSize > 0 {
		cache, _ = lru.New2Q(cacheSize)
	}

	return &BaseSuffrage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", name)
		}),
		local:          local,
		numberOfActing: numberOfActing,
		cacheSize:      cacheSize,
		cache:          cache,
		electFunc:      electFunc,
	}
}

func (sf *BaseSuffrage) Initialize() error {
	return nil
}

func (sf *BaseSuffrage) Local() *isaac.Local {
	return sf.local
}

func (sf *BaseSuffrage) Cache() *lru.TwoQueueCache {
	return sf.cache
}

func (sf *BaseSuffrage) CacheSize() int {
	return sf.cacheSize
}

func (sf *BaseSuffrage) NumberOfActing() uint {
	return sf.numberOfActing
}

func (sf *BaseSuffrage) Name() string {
	return sf.name
}

func (sf *BaseSuffrage) IsInside(a base.Address) bool {
	return sf.local.Nodes().Exists(a)
}

func (sf *BaseSuffrage) Acting(height base.Height, round base.Round) (base.ActingSuffrage, error) {
	if sf.cache == nil {
		return sf.electFunc(height, round)
	}

	cacheKey := sf.cacheKey(height, round)
	if af, found := sf.cache.Get(cacheKey); found {
		return af.(base.ActingSuffrage), nil
	}

	if af, err := sf.electFunc(height, round); err != nil {
		return af, err
	} else {
		sf.cache.Add(cacheKey, af)

		return af, nil
	}
}

func (sf *BaseSuffrage) IsActing(height base.Height, round base.Round, node base.Address) (bool, error) {
	if af, err := sf.Acting(height, round); err != nil {
		return false, err
	} else {
		return af.Exists(node), nil
	}
}

func (sf *BaseSuffrage) IsProposer(height base.Height, round base.Round, node base.Address) (bool, error) {
	if af, err := sf.Acting(height, round); err != nil {
		return false, err
	} else {
		return af.Proposer().Equal(node), nil
	}
}

func (sf *BaseSuffrage) Nodes() []base.Address {
	ns := []base.Address{sf.local.Node().Address()}
	sf.local.Nodes().Traverse(func(n network.Node) bool {
		ns = append(ns, n.Address())

		return true
	})

	return ns
}

func (sf *BaseSuffrage) cacheKey(height base.Height, round base.Round) string {
	return fmt.Sprintf("%d-%d", height.Int64(), round.Uint64())
}

func (sf *BaseSuffrage) Verbose() string {
	return ""
}
