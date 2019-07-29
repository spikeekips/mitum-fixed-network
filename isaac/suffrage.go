package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/node"
)

type Suffrage interface {
	Nodes() []node.Node
	AddNodes(...node.Node) Suffrage
	RemoveNodes(...node.Node) Suffrage
	Acting(height Height, round Round) ActingSuffrage
}

type ActingSuffrage struct {
	height   Height
	round    Round
	proposer node.Node
	nodes    []node.Node
}

func NewActingSuffrage(height Height, round Round, proposer node.Node, nodes []node.Node) ActingSuffrage {
	return ActingSuffrage{height: height, round: round, proposer: proposer, nodes: nodes}
}

func (af ActingSuffrage) Proposer() node.Node {
	return af.proposer
}

func (af ActingSuffrage) Nodes() []node.Node {
	// TODO nodes should be sorted by it's address
	return af.nodes
}

func (af ActingSuffrage) Height() Height {
	return af.height
}

func (af ActingSuffrage) Round() Round {
	return af.round
}

func (af ActingSuffrage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"height":   af.height,
		"round":    af.round,
		"proposer": af.proposer,
		"nodes":    af.nodes,
	})
}

func (af ActingSuffrage) String() string {
	b, _ := json.Marshal(af)
	return string(b)
}
