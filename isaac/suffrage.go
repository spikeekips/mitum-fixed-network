package isaac

type Suffrage interface {
	Name() string
	Acting(Height, Round) ActingSuffrage
	IsActing(Height, Round, Address /* node */) bool
	IsProposer(Height, Round, Address /* node */) bool
}

type ActingSuffrage struct {
	height   Height
	round    Round
	proposer Node
	nodes    map[Address]Node
}

func NewActingSuffrage(height Height, round Round, proposer Node, selected []Node) ActingSuffrage {
	nodes := map[Address]Node{}
	for _, n := range selected {
		nodes[n.Address()] = n
	}

	return ActingSuffrage{
		height:   height,
		round:    round,
		proposer: proposer,
		nodes:    nodes,
	}
}

func (as ActingSuffrage) Height() Height {
	return as.height
}

func (as ActingSuffrage) Round() Round {
	return as.round
}

func (as ActingSuffrage) Nodes() map[Address]Node {
	return as.nodes
}

func (as ActingSuffrage) Exists(node Address) bool {
	_, found := as.nodes[node]
	return found
}

func (as ActingSuffrage) Proposer() Node {
	return as.proposer
}
