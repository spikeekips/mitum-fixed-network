package isaac

import (
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
)

type NodesPool struct {
	sync.RWMutex
	localNode *LocalNode
	nodesMap  map[string]network.Node
	nodes     []base.Address
}

func NewNodesPool(localNode *LocalNode) *NodesPool {
	return &NodesPool{localNode: localNode, nodesMap: map[string]network.Node{}}
}

func (ns *NodesPool) Node(address base.Address) (network.Node, bool) {
	ns.RLock()
	defer ns.RUnlock()

	n, found := ns.nodesMap[address.String()]

	return n, found
}

func (ns *NodesPool) Exists(address base.Address) bool {
	ns.RLock()
	defer ns.RUnlock()

	return ns.exists(address)
}

func (ns *NodesPool) exists(address base.Address) bool {
	_, found := ns.nodesMap[address.String()]

	return found
}

func (ns *NodesPool) Add(nl ...network.Node) error {
	ns.Lock()
	defer ns.Unlock()

	founds := map[string]struct{}{}
	for _, n := range nl {
		if n.Address().Equal(ns.localNode.Address()) {
			return xerrors.Errorf("local node can not be added")
		} else if ns.exists(n.Address()) {
			return xerrors.Errorf("same Address already exists; %v", n.Address())
		} else if _, found := founds[n.Address().String()]; found {
			return xerrors.Errorf("duplicated Address found; %v", n.Address())
		} else {
			founds[n.Address().String()] = struct{}{}
		}
	}

	for _, n := range nl {
		ns.nodesMap[n.Address().String()] = n
		ns.nodes = append(ns.nodes, n.Address())
	}

	base.SortAddresses(ns.nodes)

	return nil
}

func (ns *NodesPool) Remove(addresses ...base.Address) error {
	ns.Lock()
	defer ns.Unlock()

	founds := map[string]struct{}{}
	for _, address := range addresses {
		if !ns.exists(address) {
			return xerrors.Errorf("Address does not exist; %v", address)
		} else if _, found := founds[address.String()]; found {
			return xerrors.Errorf("duplicated Address found; %v", address)
		} else {
			founds[address.String()] = struct{}{}
		}
	}

	nodes := make([]base.Address, len(ns.nodes)-len(addresses))

	var i int
	for j := range ns.nodes {
		var deleted bool
		for k := range addresses {
			if addresses[k].Equal(ns.nodes[j]) {
				deleted = true

				break
			}
		}

		if deleted {
			delete(ns.nodesMap, ns.nodes[j].String())

			continue
		}

		nodes[i] = ns.nodes[j]
		i++
	}

	base.SortAddresses(nodes)

	ns.nodes = nodes

	return nil
}

func (ns *NodesPool) Len() int {
	ns.RLock()
	defer ns.RUnlock()

	return len(ns.nodesMap)
}

// Traverse returns the sorted nodes.
func (ns *NodesPool) Traverse(callback func(network.Node) bool) {
	var nodes []network.Node
	ns.RLock()

	{
		if len(ns.nodesMap) < 1 {
			return
		}

		for i := range ns.nodes {
			nodes = append(nodes, ns.nodesMap[ns.nodes[i].String()])
		}
	}
	ns.RUnlock()

	for _, n := range nodes {
		if !callback(n) {
			break
		}
	}
}
