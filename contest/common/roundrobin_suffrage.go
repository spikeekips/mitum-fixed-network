package common

import (
	"fmt"
	"sort"
	"strings"

	lru "github.com/hashicorp/golang-lru"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/logging"
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

func (sf *RoundrobinSuffrage) IsInside(a isaac.Address) bool {
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

func (sf *RoundrobinSuffrage) cacheKey(height isaac.Height, round isaac.Round) string {
	return fmt.Sprintf("%d-%d", height.Int64(), round.Uint64())
}

func (sf *RoundrobinSuffrage) Acting(height isaac.Height, round isaac.Round) isaac.ActingSuffrage {
	if sf.cache == nil {
		return sf.acting(height, round)
	}

	cacheKey := sf.cacheKey(height, round)
	if af, found := sf.cache.Get(cacheKey); found {
		return af.(isaac.ActingSuffrage)
	}

	af := sf.acting(height, round)
	sf.cache.Add(cacheKey, af)

	return af
}

func (sf *RoundrobinSuffrage) acting(height isaac.Height, round isaac.Round) isaac.ActingSuffrage {
	all := []isaac.Node{sf.localstate.Node()}
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

	var selected []isaac.Node
	if len(all) == numberOfActingSuffrageNodes {
		selected = all
	} else {
		selected = append(selected, all[pos])
		selected = append(selected, all[pos+1:]...)

		if len(selected) > numberOfActingSuffrageNodes {
			selected = selected[:numberOfActingSuffrageNodes]
		} else if len(selected) < numberOfActingSuffrageNodes {
			selected = append(selected, all[:numberOfActingSuffrageNodes-len(selected)]...)
		}
	}

	return isaac.NewActingSuffrage(height, round, all[pos], selected)
}

func (sf *RoundrobinSuffrage) pos(height isaac.Height, round isaac.Round, all int) int {
	sum := uint64(height.Int64()) + round.Uint64()

	return int(sum % uint64(all))
}

func (sf *RoundrobinSuffrage) IsActing(height isaac.Height, round isaac.Round, node isaac.Address) bool {
	af := sf.Acting(height, round)

	return af.Exists(node)
}

func (sf *RoundrobinSuffrage) IsProposer(height isaac.Height, round isaac.Round, node isaac.Address) bool {
	af := sf.Acting(height, round)

	return af.Proposer().Address().Equal(node)
}
