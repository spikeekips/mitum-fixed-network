package common

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/logging"
)

// FixedSuffrage will be used only for testing.
type FixedSuffrage struct {
	*logging.Logging
	proposer  base.Node
	nodes     map[base.Address]base.Node
	nodeSlice []base.Node
}

func NewFixedSuffrage(proposer base.Node, nodes []base.Node) *FixedSuffrage {
	ns := map[base.Address]base.Node{}
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

func (ff *FixedSuffrage) IsInside(a base.Address) bool {
	_, found := ff.nodes[a]
	return found
}

func (ff *FixedSuffrage) Acting(height base.Height, round base.Round) base.ActingSuffrage {
	return base.NewActingSuffrage(height, round, ff.proposer, ff.nodeSlice)
}

func (ff *FixedSuffrage) IsActing(_ base.Height, _ base.Round, node base.Address) bool {
	_, found := ff.nodes[node]
	return found
}

func (ff *FixedSuffrage) IsProposer(_ base.Height, _ base.Round, node base.Address) bool {
	return ff.proposer.Address().Equal(node)
}
