package process

import (
	"fmt"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/logging"
)

type ActinfSuffrageElectFunc func(base.Height, base.Round) (base.ActingSuffrage, error)

type BaseSuffrage struct {
	sync.RWMutex
	*logging.Logging
	name           string
	nodes          []base.Address
	numberOfActing uint
	cacheSize      int
	cache          *lru.TwoQueueCache
	electFunc      ActinfSuffrageElectFunc
	nodesMap       map[string]struct{}
}

func NewBaseSuffrage(
	name string,
	nodes []base.Address,
	numberOfActing uint,
	electFunc ActinfSuffrageElectFunc,
	cacheSize int,
) (*BaseSuffrage, error) {
	if len(nodes) < int(numberOfActing) {
		return nil, errors.Errorf("nodes is under number of acting, %d < %d", len(nodes), numberOfActing)
	}

	nm := map[string]struct{}{}
	for i := range nodes {
		nm[nodes[i].String()] = struct{}{}
	}

	var cache *lru.TwoQueueCache
	if cacheSize > 0 {
		cache, _ = lru.New2Q(cacheSize)
	}

	base.SortAddresses(nodes)

	return &BaseSuffrage{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", name)
		}),
		nodes:          nodes,
		numberOfActing: numberOfActing,
		cacheSize:      cacheSize,
		cache:          cache,
		electFunc:      electFunc,
		nodesMap:       nm,
	}, nil
}

func (*BaseSuffrage) Initialize() error {
	return nil
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
	_, found := sf.nodesMap[a.String()]

	return found
}

func (sf *BaseSuffrage) Acting(height base.Height, round base.Round) (base.ActingSuffrage, error) {
	if sf.cache == nil {
		return sf.electFunc(height, round)
	}

	cacheKey := sf.cacheKey(height, round)
	if af, found := sf.cache.Get(cacheKey); found {
		return af.(base.ActingSuffrage), nil
	}

	af, err := sf.electFunc(height, round)
	if err != nil {
		return af, err
	}
	sf.cache.Add(cacheKey, af)

	return af, nil
}

func (sf *BaseSuffrage) IsActing(height base.Height, round base.Round, n base.Address) (bool, error) {
	af, err := sf.Acting(height, round)
	if err != nil {
		return false, err
	}
	return af.Exists(n), nil
}

func (sf *BaseSuffrage) IsProposer(height base.Height, round base.Round, n base.Address) (bool, error) {
	af, err := sf.Acting(height, round)
	if err != nil {
		return false, err
	}
	return af.Proposer().Equal(n), nil
}

func (sf *BaseSuffrage) Nodes() []base.Address {
	return sf.nodes
}

func (*BaseSuffrage) cacheKey(height base.Height, round base.Round) string {
	return fmt.Sprintf("%d-%d", height.Int64(), round.Uint64())
}

func (*BaseSuffrage) Verbose() string {
	return ""
}
