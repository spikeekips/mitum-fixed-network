package isaac

import (
	"sync"

	"golang.org/x/xerrors"
)

type NodesState struct {
	sync.RWMutex
	localNode *LocalNode
	nodes     map[Address]Node
}

func NewNodesState(localNode *LocalNode, nodes []Node) *NodesState {
	m := map[Address]Node{}
	for _, n := range nodes {
		if n.Address().Equal(localNode.Address()) {
			continue
		}
		if _, found := m[n.Address()]; found {
			continue
		}
		m[n.Address()] = n
	}

	return &NodesState{localNode: localNode, nodes: m}
}

func (ns *NodesState) Node(address Address) (Node, bool) {
	ns.RLock()
	defer ns.RUnlock()
	n, found := ns.nodes[address]

	return n, found
}

func (ns *NodesState) Exists(address Address) bool {
	ns.RLock()
	defer ns.RUnlock()

	return ns.exists(address)
}

func (ns *NodesState) exists(address Address) bool {
	_, found := ns.nodes[address]

	return found
}

func (ns *NodesState) Add(nl ...Node) error {
	ns.Lock()
	defer ns.Unlock()

	for _, n := range nl {
		if n.Address().Equal(ns.localNode.Address()) {
			return xerrors.Errorf("local node can be added")
		}

		if ns.exists(n.Address()) {
			return xerrors.Errorf("same Address already exists; %v", n.Address())
		}
	}

	for _, n := range nl {
		ns.nodes[n.Address()] = n
	}

	return nil
}

func (ns *NodesState) Remove(addresses ...Address) error {
	ns.Lock()
	defer ns.Unlock()

	for _, address := range addresses {
		if !ns.exists(address) {
			return xerrors.Errorf("Address does not exist; %v", address)
		}
	}

	for _, address := range addresses {
		delete(ns.nodes, address)
	}

	return nil
}

func (ns *NodesState) Len() int {
	return len(ns.nodes)
}

func (ns *NodesState) Traverse(callback func(Node) bool) {
	var nodes []Node
	ns.RLock()
	{
		if len(ns.nodes) < 1 {
			return
		}

		for _, n := range ns.nodes {
			nodes = append(nodes, n)
		}
	}
	ns.RUnlock()

	for _, n := range nodes {
		if !callback(n) {
			break
		}
	}
}
