package contestlib

import (
	"fmt"
	"sort"
	"strings"

	lru "github.com/hashicorp/golang-lru"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/logging"
)

type RoundrobinSuffrage struct {
	*logging.Logging
	localstate *isaac.Localstate
	cache      *lru.TwoQueueCache
}

func NewRoundrobinSuffrage(localstate *isaac.Localstate, cacheSize int) *RoundrobinSuffrage {
	var cache *lru.TwoQueueCache
	if cacheSize > 0 {
		cache, _ = lru.New2Q(cacheSize)
	}

	return &RoundrobinSuffrage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "roundrobin-suffrage")
		}),
		localstate: localstate,
		cache:      cache,
	}
}

func (sf *RoundrobinSuffrage) Name() string {
	return "roundrobin-suffrage"
}

func (sf *RoundrobinSuffrage) IsInside(a base.Address) bool {
	var found bool
	sf.localstate.Nodes().Traverse(func(n isaac.Node) bool {
		if n.Address().Equal(a) {
			found = true
			return false
		}
		return true
	})

	return found
}

func (sf *RoundrobinSuffrage) cacheKey(height base.Height, round base.Round) string {
	return fmt.Sprintf("%d-%d", height.Int64(), round.Uint64())
}

func (sf *RoundrobinSuffrage) Acting(height base.Height, round base.Round) base.ActingSuffrage {
	if sf.cache == nil {
		return sf.acting(height, round)
	}

	cacheKey := sf.cacheKey(height, round)
	if af, found := sf.cache.Get(cacheKey); found {
		return af.(base.ActingSuffrage)
	}

	af := sf.acting(height, round)
	sf.cache.Add(cacheKey, af)

	return af
}

func (sf *RoundrobinSuffrage) acting(height base.Height, round base.Round) base.ActingSuffrage {
	all := []base.Node{sf.localstate.Node()}
	sf.localstate.Nodes().Traverse(func(n isaac.Node) bool {
		all = append(all, n)

		return true
	})

	if len(all) > 1 {
		sort.Slice(
			all,
			func(i, j int) bool {
				return strings.Compare(
					all[i].Address().String(),
					all[j].Address().String(),
				) < 0
			},
		)
	}

	numberOfActingSuffrageNodes := int(sf.localstate.Policy().NumberOfActingSuffrageNodes())
	if len(all) < numberOfActingSuffrageNodes {
		numberOfActingSuffrageNodes = len(all)
	}

	pos := sf.pos(height, round, len(all))

	var selected []base.Address
	if len(all) == numberOfActingSuffrageNodes {
		for _, n := range all {
			selected = append(selected, n.Address())
		}
	} else {
		selected = append(selected, all[pos].Address())

		for _, n := range all[pos+1:] {
			selected = append(selected, n.Address())
		}

		if len(selected) > numberOfActingSuffrageNodes {
			selected = selected[:numberOfActingSuffrageNodes]
		} else if len(selected) < numberOfActingSuffrageNodes {
			for _, n := range all[:numberOfActingSuffrageNodes-len(selected)] {
				selected = append(selected, n.Address())
			}
		}
	}

	return base.NewActingSuffrage(height, round, all[pos].Address(), selected)
}

func (sf *RoundrobinSuffrage) pos(height base.Height, round base.Round, all int) int {
	sum := uint64(height.Int64()) + round.Uint64()

	return int(sum % uint64(all))
}

func (sf *RoundrobinSuffrage) IsActing(height base.Height, round base.Round, node base.Address) bool {
	af := sf.Acting(height, round)

	return af.Exists(node)
}

func (sf *RoundrobinSuffrage) IsProposer(height base.Height, round base.Round, node base.Address) bool {
	af := sf.Acting(height, round)

	return af.Proposer().Equal(node)
}

func (sf *RoundrobinSuffrage) Nodes() []base.Address {
	var ns []base.Address
	sf.localstate.Nodes().Traverse(func(n isaac.Node) bool {
		ns = append(ns, n.Address())

		return true
	})

	return ns
}
