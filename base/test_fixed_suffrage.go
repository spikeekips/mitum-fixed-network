// +build test

package base

import (
	"github.com/spikeekips/mitum/util/logging"
)

// FixedSuffrage will be used only for testing.
type FixedSuffrage struct {
	*logging.Logging
	proposer Address
	nodes    map[Address]struct{}
	nodeList []Address
}

func NewFixedSuffrage(proposer Node, nodes []Node) *FixedSuffrage {
	ns := map[Address]struct{}{}
	var nodeList []Address
	for _, n := range nodes {
		ns[n.Address()] = struct{}{}
		nodeList = append(nodeList, n.Address())
	}

	return &FixedSuffrage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "fixed-suffrage")
		}),
		proposer: proposer.Address(),
		nodes:    ns,
		nodeList: nodeList,
	}
}

func (ff *FixedSuffrage) Name() string {
	return "fixed-suffrage"
}

func (ff *FixedSuffrage) IsInside(a Address) bool {
	_, found := ff.nodes[a]
	return found
}

func (ff *FixedSuffrage) Acting(height Height, round Round) ActingSuffrage {
	return NewActingSuffrage(height, round, ff.proposer, ff.nodeList)
}

func (ff *FixedSuffrage) IsActing(_ Height, _ Round, node Address) bool {
	_, found := ff.nodes[node]
	return found
}

func (ff *FixedSuffrage) IsProposer(_ Height, _ Round, node Address) bool {
	return ff.proposer.Equal(node)
}
