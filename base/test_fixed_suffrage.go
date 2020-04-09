// +build test

package base

import (
	"github.com/spikeekips/mitum/util/logging"
)

// FixedSuffrage will be used only for testing.
type FixedSuffrage struct {
	*logging.Logging
	proposer  Node
	nodes     map[Address]Node
	nodeSlice []Node
}

func NewFixedSuffrage(proposer Node, nodes []Node) *FixedSuffrage {
	ns := map[Address]Node{}
	for _, n := range nodes {
		ns[n.Address()] = n
	}

	return &FixedSuffrage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "fixed-suffrage")
		}),
		proposer:  proposer,
		nodes:     ns,
		nodeSlice: nodes,
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
	return NewActingSuffrage(height, round, ff.proposer, ff.nodeSlice)
}

func (ff *FixedSuffrage) IsActing(_ Height, _ Round, node Address) bool {
	_, found := ff.nodes[node]
	return found
}

func (ff *FixedSuffrage) IsProposer(_ Height, _ Round, node Address) bool {
	return ff.proposer.Address().Equal(node)
}
