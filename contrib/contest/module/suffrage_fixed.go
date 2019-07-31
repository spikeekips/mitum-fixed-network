package contest_module

import (
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type FixedProposerSuffrage struct {
	proposer node.Node
	nodes    []node.Node
}

func NewFixedProposerSuffrage(proposer node.Node, nodes ...node.Node) FixedProposerSuffrage {
	return FixedProposerSuffrage{proposer: proposer, nodes: nodes}
}

func (fs FixedProposerSuffrage) AddNodes(nodes ...node.Node) isaac.Suffrage {
	fs.nodes = append(fs.nodes, nodes...)
	return fs
}

func (fs FixedProposerSuffrage) RemoveNodes(nodes ...node.Node) isaac.Suffrage {
	var newNodes []node.Node
	for _, a := range fs.nodes {
		var found bool
		for _, b := range nodes {
			if a.Equal(b) {
				found = true
				break
			}
		}

		if found {
			continue
		}
		newNodes = append(newNodes, a)
	}

	fs.nodes = newNodes
	return fs
}

func (fs FixedProposerSuffrage) Nodes() []node.Node {
	return fs.nodes
}

func (fs FixedProposerSuffrage) Acting(height isaac.Height, round isaac.Round) isaac.ActingSuffrage {
	return isaac.NewActingSuffrage(height, round, fs.proposer, fs.nodes)
}
