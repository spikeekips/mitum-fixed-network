// +build test

package isaac

import "github.com/spikeekips/mitum/node"

type FixedProposerSuffrage struct {
	proposer node.Node
	nodes    []node.Node
}

func NewFixedProposerSuffrage(proposer node.Node, nodes ...node.Node) FixedProposerSuffrage {
	return FixedProposerSuffrage{proposer: proposer, nodes: nodes}
}

func (fs FixedProposerSuffrage) NumberOfActing() uint {
	return uint(len(fs.nodes))
}

func (fs FixedProposerSuffrage) AddNodes(nodes ...node.Node) Suffrage {
	fs.nodes = append(fs.nodes, nodes...)
	return fs
}

func (fs FixedProposerSuffrage) RemoveNodes(nodes ...node.Node) Suffrage {
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

func (fs FixedProposerSuffrage) Acting(height Height, round Round) ActingSuffrage {
	return NewActingSuffrage(height, round, fs.proposer, fs.nodes)
}

func (fs FixedProposerSuffrage) Exists(_ Height, address node.Address) bool {
	for _, n := range fs.nodes {
		if n.Address().Equal(address) {
			return true
		}
	}

	return false
}
