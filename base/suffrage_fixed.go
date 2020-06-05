package base

import (
	"github.com/spikeekips/mitum/util/logging"
)

// FixedSuffrage will be used for creating genesis block or testing.
type FixedSuffrage struct {
	*logging.Logging
	proposer Address
	nodes    map[Address]struct{}
	nodeList []Address
}

func NewFixedSuffrage(proposer Address, nodes []Address) *FixedSuffrage {
	ns := map[Address]struct{}{
		proposer: {},
	}
	nodeList := []Address{proposer}
	for _, n := range nodes {
		if _, found := ns[n]; found {
			continue
		}

		ns[n] = struct{}{}
		nodeList = append(nodeList, n)
	}

	return &FixedSuffrage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "fixed-suffrage")
		}),
		proposer: proposer,
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

func (ff *FixedSuffrage) Nodes() []Address {
	return ff.nodeList
}
