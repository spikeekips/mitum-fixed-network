package mitum

// FixedSuffrage will be used only for testing.
type FixedSuffrage struct {
	proposer Node
	nodes    map[Address]Node
}

func NewFixedSuffrage(proposer Node, nodes []Node) *FixedSuffrage {
	ns := map[Address]Node{}
	for _, n := range nodes {
		ns[n.Address()] = n
	}

	return &FixedSuffrage{
		proposer: proposer,
		nodes:    ns,
	}
}

func (ff *FixedSuffrage) Name() string {
	return "fixed-suffrage"
}

func (ff *FixedSuffrage) Acting(height Height, round Round) ActingSuffrage {
	return ActingSuffrage{
		height:   height,
		round:    round,
		proposer: ff.proposer,
		nodes:    ff.nodes,
	}
}

func (ff *FixedSuffrage) IsActing(_ Height, _ Round, node Address) bool {
	_, found := ff.nodes[node]
	return found
}

func (ff *FixedSuffrage) IsProposer(_ Height, _ Round, node Address) bool {
	return ff.proposer.Address().Equal(node)
}
