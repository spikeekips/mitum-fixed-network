package contest_module

import (
	"sync"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

func init() {
	Suffrages = append(Suffrages, "FixedProposerSuffrage")
}

type FixedProposerSuffrage struct {
	sync.RWMutex
	proposer node.Node
	nodes    []node.Node
}

func NewFixedProposerSuffrage(proposer node.Node, nodes ...node.Node) *FixedProposerSuffrage {
	return &FixedProposerSuffrage{proposer: proposer, nodes: nodes}
}

func (fs FixedProposerSuffrage) AddNodes(nodes ...node.Node) isaac.Suffrage {
	fs.Lock()
	defer fs.Unlock()

	fs.nodes = append(fs.nodes, nodes...)
	return fs
}

func (fs FixedProposerSuffrage) RemoveNodes(nodes ...node.Node) isaac.Suffrage {
	fs.Lock()
	defer fs.Unlock()

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
	fs.RLock()
	defer fs.RUnlock()

	return fs.nodes
}

func (fs FixedProposerSuffrage) Acting(height isaac.Height, round isaac.Round) isaac.ActingSuffrage {
	fs.RLock()
	defer fs.RUnlock()

	return isaac.NewActingSuffrage(height, round, fs.proposer, fs.nodes)
}
