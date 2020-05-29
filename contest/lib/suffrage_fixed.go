package contestlib

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/logging"
)

type FixedSuffrage struct {
	*logging.Logging
	proposer base.Address
	nodes    map[base.Address]struct{}
	nodeList []base.Address
}

func NewFixedSuffrage(proposer base.Address, nodes []base.Address) *FixedSuffrage {
	ns := map[base.Address]struct{}{}
	for _, a := range nodes {
		ns[a] = struct{}{}
	}
	if _, found := ns[proposer]; !found {
		ns[proposer] = struct{}{}
		nodes = append(nodes, proposer)
	}

	return &FixedSuffrage{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "fixed-suffrage")
		}),
		proposer: proposer,
		nodes:    ns,
		nodeList: nodes,
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
	return base.NewActingSuffrage(height, round, ff.proposer, ff.nodeList)
}

func (ff *FixedSuffrage) IsActing(_ base.Height, _ base.Round, node base.Address) bool {
	_, found := ff.nodes[node]
	return found
}

func (ff *FixedSuffrage) IsProposer(_ base.Height, _ base.Round, node base.Address) bool {
	return ff.proposer.Equal(node)
}
