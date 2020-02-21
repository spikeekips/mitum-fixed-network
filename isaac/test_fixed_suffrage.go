// +build test

package isaac

import (
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/logging"
)

// FixedSuffrage will be used only for testing.
type FixedSuffrage struct {
	*logging.Logger
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
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
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
