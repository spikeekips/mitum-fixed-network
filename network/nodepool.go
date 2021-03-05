package network

import (
	"sync"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
)

// Nodepool contains all the known nodes including local node.
type Nodepool struct {
	sync.RWMutex
	local    *LocalNode
	nodesMap map[string]Node
}

func NewNodepool(local *LocalNode) *Nodepool {
	return &Nodepool{
		local: local,
		nodesMap: map[string]Node{
			local.Address().String(): local,
		},
	}
}

func (np *Nodepool) Node(address base.Address) (Node, bool) {
	np.RLock()
	defer np.RUnlock()

	n, found := np.nodesMap[address.String()]

	return n, found
}

func (np *Nodepool) Local() *LocalNode {
	return np.local
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
		if np.exists(n.Address()) {
			return xerrors.Errorf("same Address already exists; %v", n.Address())
		} else if _, found := founds[n.Address().String()]; found {
			return xerrors.Errorf("duplicated Address found; %v", n.Address())
		} else {
			founds[n.Address().String()] = struct{}{}
		}
	}

	for _, n := range nl {
		np.nodesMap[n.Address().String()] = n
	}

	return nil
}

func (np *Nodepool) Remove(addresses ...base.Address) error {
	np.Lock()
	defer np.Unlock()

	founds := map[string]struct{}{}
	for _, address := range addresses {
		if address.Equal(np.local.Address()) {
			return xerrors.Errorf("local can not be removed; %v", address)
		}

		if !np.exists(address) {
			return xerrors.Errorf("Address does not exist; %v", address)
		} else if _, found := founds[address.String()]; found {
			return xerrors.Errorf("duplicated Address found; %v", address)
		} else {
			founds[address.String()] = struct{}{}
		}
	}

	for i := range addresses {
		delete(np.nodesMap, addresses[i].String())
	}

	return nil
}

func (np *Nodepool) Len() int {
	np.RLock()
	defer np.RUnlock()

	return len(np.nodesMap)
}

func (np *Nodepool) Traverse(callback func(Node) bool) {
	nodes := make([]Node, len(np.nodesMap))
	np.RLock()
	if len(np.nodesMap) < 1 {
		return
	}

	var i int
	for k := range np.nodesMap {
		nodes[i] = np.nodesMap[k]
		i++
	}
	np.RUnlock()

	for _, n := range nodes {
		if !callback(n) {
			break
		}
	}
}

func (np *Nodepool) TraverseRemotes(callback func(Node) bool) {
	np.RLock()
	if len(np.nodesMap) < 2 {
		return
	}
	np.RUnlock()

	np.Traverse(func(n Node) bool {
		if n.Address().Equal(np.local.Address()) {
			return true
		}

		return callback(n)
	})
}

func (np *Nodepool) Addresses() []base.Address {
	nodes := make([]base.Address, np.Len())

	var i int
	np.Traverse(func(n Node) bool {
		nodes[i] = n.Address()
		i++

		return true
	})

	return nodes
}
