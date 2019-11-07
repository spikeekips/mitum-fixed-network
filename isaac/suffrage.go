package isaac

import (
	"encoding/json"

	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/node"
)

type Suffrage interface {
	Nodes() []node.Node
	NumberOfActing() uint
	Acting(height Height, round Round) ActingSuffrage
	Exists(height Height, address node.Address) bool
	AddNodes(...node.Node) Suffrage
	RemoveNodes(...node.Node) Suffrage
}

type ActingSuffrage struct {
	height   Height
	round    Round
	proposer node.Node
	nodes    []node.Node
}

func NewActingSuffrage(height Height, round Round, proposer node.Node, nodes []node.Node) ActingSuffrage {
	node.SortNodesByAddress(nodes)

	return ActingSuffrage{height: height, round: round, proposer: proposer, nodes: nodes}
}

func (af ActingSuffrage) Proposer() node.Node {
	return af.proposer
}

func (af ActingSuffrage) Nodes() []node.Node {
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

func (af ActingSuffrage) MarshalZerologObject(e *zerolog.Event) {
	e.Uint64("height", af.height.Uint64())
	e.Uint64("round", af.round.Uint64())
	e.Object("proposer", af.proposer)

	ns := zerolog.Arr()
	for _, n := range af.nodes {
		ns.Object(n)
	}

	e.Array("nodes", ns)
}

func (af ActingSuffrage) String() string {
	b, _ := json.Marshal(af) // nolint
	return string(b)
}

func (af ActingSuffrage) Exists(address node.Address) bool {
	for _, n := range af.nodes {
		if n.Address() == address {
			return true
		}
	}

	return false
}
