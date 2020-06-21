package base

import "github.com/spikeekips/mitum/util"

type Suffrage interface {
	util.Initializer
	Name() string
	Acting(Height, Round) ActingSuffrage
	IsInside(Address) bool
	IsActing(Height, Round, Address /* node address */) bool
	IsProposer(Height, Round, Address /* node address */) bool
	Nodes() []Address
}

type ActingSuffrage struct {
	height   Height
	round    Round
	proposer Address
	nodes    map[Address]struct{}
	nodeList []Address
}

func NewActingSuffrage(height Height, round Round, proposer Address, selected []Address) ActingSuffrage {
	nodes := map[Address]struct{}{}
	for _, n := range selected {
		nodes[n] = struct{}{}
	}

	return ActingSuffrage{
		height:   height,
		round:    round,
		proposer: proposer,
		nodes:    nodes,
		nodeList: selected,
	}
}

func (as ActingSuffrage) Height() Height {
	return as.height
}

func (as ActingSuffrage) Round() Round {
	return as.round
}

func (as ActingSuffrage) Nodes() []Address {
	return as.nodeList
}

func (as ActingSuffrage) Exists(node Address) bool {
	_, found := as.nodes[node]
	return found
}

func (as ActingSuffrage) Proposer() Address {
	return as.proposer
}
