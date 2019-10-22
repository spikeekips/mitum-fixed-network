package contest_module

import (
	"encoding/json"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type FixedProposerSuffrage struct {
	proposer       node.Node
	numberOfActing uint // by default numberOfActing is 0; it means all nodes will be acting member
	nodes          []node.Node
	others         []node.Node
}

func NewFixedProposerSuffrage(proposer node.Node, numberOfActing uint, nodes ...node.Node) *FixedProposerSuffrage {
	sorted := make([]node.Node, len(nodes))
	copy(sorted, nodes)

	node.SortNodesByAddress(sorted)

	if int(numberOfActing) > len(sorted) {
		panic(xerrors.Errorf(
			"numberOfActing should be lesser than number of nodes: numberOfActing=%v nodes=%v",
			numberOfActing,
			len(sorted),
		))
	}

	var others []node.Node
	for _, n := range sorted {
		if n.Address().Equal(proposer.Address()) {
			continue
		}
		others = append(others, n)
	}

	return &FixedProposerSuffrage{
		proposer:       proposer,
		numberOfActing: numberOfActing,
		nodes:          sorted,
		others:         others,
	}
}

func (fs FixedProposerSuffrage) AddNodes(_ ...node.Node) isaac.Suffrage {
	return fs
}

func (fs FixedProposerSuffrage) RemoveNodes(_ ...node.Node) isaac.Suffrage {
	return fs
}

func (fs FixedProposerSuffrage) Nodes() []node.Node {
	return fs.nodes
}

func (fs FixedProposerSuffrage) Acting(height isaac.Height, round isaac.Round) isaac.ActingSuffrage {
	var nodes []node.Node
	if fs.numberOfActing == 0 || int(fs.numberOfActing) == len(nodes) {
		nodes = fs.nodes
	} else {
		nodes = append(nodes, fs.proposer)
		nodes = append(
			nodes,
			selectNodes(height, round, int(fs.numberOfActing)-1, fs.others)...,
		)
	}

	return isaac.NewActingSuffrage(height, round, fs.proposer, nodes)
}

func (fs FixedProposerSuffrage) Exists(_ isaac.Height, address node.Address) bool {
	for _, n := range fs.nodes {
		if n.Address().Equal(address) {
			return true
		}
	}

	return false
}

func (fs FixedProposerSuffrage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":             "FixedProposerSuffrage",
		"proposer":         fs.proposer,
		"nodes":            fs.nodes,
		"number_of_acting": fs.numberOfActing,
	})
}

func selectNodes(height isaac.Height, round isaac.Round, n int, nodes []node.Node) []node.Node {
	if n == 0 || n == len(nodes) {
		return nodes
	}

	var selected []node.Node
	index := (height.Int64() + int64(round)) % int64(len(nodes))
	selected = append(selected, nodes[index:]...)
	if len(selected) < n {
		selected = append(selected, nodes[:n-len(selected)]...)
	} else if len(selected) > n {
		selected = selected[:n]
	}

	return selected
}
