package contest_module

import (
	"sync"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

func init() {
	Suffrages = append(Suffrages, "RoundrobinSuffrage")
}

type RoundrobinSuffrage struct {
	sync.RWMutex
	numberOfActing uint // by default numberOfActing is 0; it means all nodes will be acting member
	nodes          []node.Node
}

func NewRoundrobinSuffrage(numberOfActing uint, nodes ...node.Node) *RoundrobinSuffrage {
	return &RoundrobinSuffrage{numberOfActing: numberOfActing, nodes: nodes}
}

func (fs *RoundrobinSuffrage) AddNodes(nodes ...node.Node) isaac.Suffrage {
	return fs
}

func (fs *RoundrobinSuffrage) RemoveNodes(nodes ...node.Node) isaac.Suffrage {
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

	nodes := selectNodes(height, round, int(fs.numberOfActing), fs.nodes)

	return isaac.NewActingSuffrage(height, round, nodes[0], nodes)
}

func (fs RoundrobinSuffrage) Exists(_ isaac.Height, _ isaac.Round, address node.Address) bool {
	fs.RLock()
	defer fs.RUnlock()

	for _, n := range fs.nodes {
		if n.Address().Equal(address) {
			return true
		}
	}

	return false
}
