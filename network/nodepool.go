package network

import (
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
)

type Nodepool struct {
	sync.RWMutex
	local    Node
	nodesMap map[string]Node
	nodes    []base.Address
}

func NewNodepool(local Node) *Nodepool {
	return &Nodepool{local: local, nodesMap: map[string]Node{}}
}

func (np *Nodepool) Node(address base.Address) (Node, bool) {
	np.RLock()
	defer np.RUnlock()

	n, found := np.nodesMap[address.String()]

	return n, found
}

func (np *Nodepool) Exists(address base.Address) bool {
	np.RLock()
	defer np.RUnlock()

	return np.exists(address)
}

func (np *Nodepool) exists(address base.Address) bool {
	_, found := np.nodesMap[address.String()]

	return found
}

func (np *Nodepool) Add(nl ...Node) error {
	np.Lock()
	defer np.Unlock()

	founds := map[string]struct{}{}
	for _, n := range nl {
		if n.Address().Equal(np.local.Address()) {
			return xerrors.Errorf("local node can not be added")
		} else if np.exists(n.Address()) {
			return xerrors.Errorf("same Address already exists; %v", n.Address())
		} else if _, found := founds[n.Address().String()]; found {
			return xerrors.Errorf("duplicated Address found; %v", n.Address())
		} else {
			founds[n.Address().String()] = struct{}{}
		}
	}

	for _, n := range nl {
		np.nodesMap[n.Address().String()] = n
		np.nodes = append(np.nodes, n.Address())
	}

	base.SortAddresses(np.nodes)

	return nil
}

func (np *Nodepool) Remove(addresses ...base.Address) error {
	np.Lock()
	defer np.Unlock()

	founds := map[string]struct{}{}
	for _, address := range addresses {
		if !np.exists(address) {
			return xerrors.Errorf("Address does not exist; %v", address)
		} else if _, found := founds[address.String()]; found {
			return xerrors.Errorf("duplicated Address found; %v", address)
		} else {
			founds[address.String()] = struct{}{}
		}
	}

	nodes := make([]base.Address, len(np.nodes)-len(addresses))

	var i int
	for j := range np.nodes {
		var deleted bool
		for k := range addresses {
			if addresses[k].Equal(np.nodes[j]) {
				deleted = true

				break
			}
		}

		if deleted {
			delete(np.nodesMap, np.nodes[j].String())

			continue
		}

		nodes[i] = np.nodes[j]
		i++
	}

	base.SortAddresses(nodes)

	np.nodes = nodes

	return nil
}

func (np *Nodepool) Len() int {
	np.RLock()
	defer np.RUnlock()

	return len(np.nodesMap)
}

// Traverse returns the sorted nodes.
func (np *Nodepool) Traverse(callback func(Node) bool) {
	var nodes []Node
	np.RLock()

	{
		if len(np.nodesMap) < 1 {
			return
		}

		for i := range np.nodes {
			nodes = append(nodes, np.nodesMap[np.nodes[i].String()])
		}
	}
	np.RUnlock()

	for _, n := range nodes {
		if !callback(n) {
			break
		}
	}
}
