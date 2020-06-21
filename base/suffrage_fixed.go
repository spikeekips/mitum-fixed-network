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
	return &FixedSuffrage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "fixed-suffrage")
		}),
		proposer: proposer,
		nodeList: nodes,
	}
}

func (ff *FixedSuffrage) Initialize() error {
	ns := map[Address]struct{}{
		ff.proposer: {},
	}
	nodeList := []Address{ff.proposer}
	for _, n := range ff.nodeList {
		if _, found := ns[n]; found {
			continue
		}

		ns[n] = struct{}{}
		nodeList = append(nodeList, n)
	}

	ff.nodes = ns
	ff.nodeList = nodeList

	return nil
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
