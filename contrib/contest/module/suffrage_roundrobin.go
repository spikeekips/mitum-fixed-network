package contest_module

import (
	"sync"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type RoundrobinSuffrage struct {
	sync.RWMutex
	nodes []node.Node
}

func NewRoundrobinSuffrage(nodes ...node.Node) *RoundrobinSuffrage {
	return &RoundrobinSuffrage{nodes: nodes}
}

func (fs *RoundrobinSuffrage) AddNodes(nodes ...node.Node) isaac.Suffrage {
	fs.Lock()
	defer fs.Unlock()

	fs.nodes = append(fs.nodes, nodes...)
	return fs
}

func (fs *RoundrobinSuffrage) RemoveNodes(nodes ...node.Node) isaac.Suffrage {
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

func (fs RoundrobinSuffrage) Nodes() []node.Node {
	fs.RLock()
	defer fs.RUnlock()

	return fs.nodes
}

func (fs RoundrobinSuffrage) Acting(height isaac.Height, round isaac.Round) isaac.ActingSuffrage {
	fs.RLock()
	defer fs.RUnlock()

	idx := (height.Int64() + int64(round)) % int64(len(fs.nodes))
	return isaac.NewActingSuffrage(height, round, fs.nodes[idx], fs.nodes)
}
