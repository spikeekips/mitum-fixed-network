package isaac

import (
	"fmt"
	"sort"
	"strings"

	lru "github.com/hashicorp/golang-lru"
)

type RoundrobinSuffrage struct {
	localState *LocalState
	cache      *lru.TwoQueueCache
}

func NewRoundrobinSuffrage(localState *LocalState, cacheSize int) *RoundrobinSuffrage {
	var cache *lru.TwoQueueCache
	if cacheSize > 0 {
		cache, _ = lru.New2Q(cacheSize)
	}

	return &RoundrobinSuffrage{
		localState: localState,
		cache:      cache,
	}
}

func (sf *RoundrobinSuffrage) Name() string {
	return "roundrobin-suffrage"
}

func (sf *RoundrobinSuffrage) cacheKey(height Height, round Round) string {
	return fmt.Sprintf("%d-%d", height.Int64(), round.Uint64())
}

func (sf *RoundrobinSuffrage) Acting(height Height, round Round) ActingSuffrage {
	if sf.cache == nil {
		return sf.acting(height, round)
	}

	cacheKey := sf.cacheKey(height, round)
	if af, found := sf.cache.Get(cacheKey); found {
		return af.(ActingSuffrage)
	}

	af := sf.acting(height, round)
	sf.cache.Add(cacheKey, af)

	return af
}

func (sf *RoundrobinSuffrage) acting(height Height, round Round) ActingSuffrage {
	all := []Node{sf.localState.Node()}
	sf.localState.Nodes().Traverse(func(n Node) bool {
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

	numberOfActingSuffrageNodes := int(sf.localState.Policy().NumberOfActingSuffrageNodes())
	if len(all) < numberOfActingSuffrageNodes {
		numberOfActingSuffrageNodes = len(all)
	}

	pos := sf.pos(height, round, len(all))

	var selected []Node
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

	nodes := map[Address]Node{}
	for _, n := range selected {
		nodes[n.Address()] = n
	}

	return ActingSuffrage{
		height:   height,
		round:    round,
		proposer: all[pos],
		nodes:    nodes,
	}
}

func (sf *RoundrobinSuffrage) pos(height Height, round Round, all int) int {
	sum := uint64(height.Int64()) + round.Uint64()

	return int(sum % uint64(all))
}

func (sf *RoundrobinSuffrage) IsActing(height Height, round Round, node Address) bool {
	af := sf.Acting(height, round)

	return af.Exists(node)
}

func (sf *RoundrobinSuffrage) IsProposer(height Height, round Round, node Address) bool {
	af := sf.Acting(height, round)

	return af.Proposer().Address().Equal(node)
}
