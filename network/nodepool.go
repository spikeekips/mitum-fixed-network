package network

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/util"
)

// Nodepool contains all the known nodes including local node.
type Nodepool struct {
	sync.RWMutex
	local   *node.Local
	localch Channel
	nodes   map[string]base.Node
	chs     map[string]Channel
}

func NewNodepool(local *node.Local, ch Channel) *Nodepool {
	addr := local.Address().String()
	return &Nodepool{
		local:   local,
		localch: ch,
		nodes: map[string]base.Node{
			addr: local,
		},
		chs: map[string]Channel{
			addr: ch,
		},
	}
}

func (np *Nodepool) Node(address base.Address) (base.Node, Channel, bool) {
	np.RLock()
	defer np.RUnlock()

	addr := address.String()
	n, found := np.nodes[addr]
	if !found {
		return nil, nil, false
	}

	return n, np.chs[addr], found
}

func (np *Nodepool) LocalNode() *node.Local {
	return np.local
}

func (np *Nodepool) LocalChannel() Channel {
	return np.localch
}

func (np *Nodepool) Exists(address base.Address) bool {
	np.RLock()
	defer np.RUnlock()

	return np.exists(address)
}

func (np *Nodepool) Add(no base.Node, ch Channel) error {
	np.Lock()
	defer np.Unlock()

	addr := no.Address().String()
	if _, found := np.nodes[addr]; found {
		return util.FoundError.Errorf("already exists")
	}

	np.nodes[addr] = no
	np.chs[addr] = ch

	return nil
}

func (np *Nodepool) SetChannel(addr base.Address, ch Channel) error {
	np.Lock()
	defer np.Unlock()

	if _, found := np.nodes[addr.String()]; !found {
		return util.NotFoundError.Errorf("unknown node, %q", addr)
	}

	np.chs[addr.String()] = ch

	if addr.Equal(np.local.Address()) {
		np.localch = ch
	}

	return nil
}

func (np *Nodepool) Remove(addrs ...base.Address) error {
	np.Lock()
	defer np.Unlock()

	founds := map[string]struct{}{}
	for _, addr := range addrs {
		if addr.Equal(np.local.Address()) {
			return errors.Errorf("local can not be removed, %q", addr)
		}

		if !np.exists(addr) {
			return errors.Errorf("Address does not exist, %q", addr)
		} else if _, found := founds[addr.String()]; found {
			return errors.Errorf("duplicated Address found, %q", addr)
		} else {
			founds[addr.String()] = struct{}{}
		}
	}

	for i := range addrs {
		addr := addrs[i].String()
		delete(np.nodes, addr)
		delete(np.chs, addr)
	}

	return nil
}

func (np *Nodepool) Len() int {
	np.RLock()
	defer np.RUnlock()

	return len(np.nodes)
}

func (np *Nodepool) LenRemoteAlives() int {
	var i int
	np.TraverseAliveRemotes(func(base.Node, Channel) bool {
		i++

		return true
	})

	return i
}

func (np *Nodepool) Traverse(callback func(base.Node, Channel) bool) {
	nodes, channels := np.nc(false)

	for i := range nodes {
		if !callback(nodes[i], channels[i]) {
			break
		}
	}
}

func (np *Nodepool) TraverseRemotes(callback func(base.Node, Channel) bool) {
	nodes, channels := np.nc(true)

	for i := range nodes {
		if !callback(nodes[i], channels[i]) {
			break
		}
	}
}

func (np *Nodepool) TraverseAliveRemotes(callback func(base.Node, Channel) bool) {
	nodes, channels := np.nc(true)

	for i := range nodes {
		ch := channels[i]
		if ch == nil {
			continue
		}

		if !callback(nodes[i], ch) {
			break
		}
	}
}

func (np *Nodepool) Addresses() []base.Address {
	nodes := make([]base.Address, np.Len())

	var i int
	np.Traverse(func(n base.Node, _ Channel) bool {
		nodes[i] = n.Address()
		i++

		return true
	})

	return nodes
}

func (np *Nodepool) exists(address base.Address) bool {
	_, found := np.nodes[address.String()]

	return found
}

func (np *Nodepool) nc(filterLocal bool) ([]base.Node, []Channel) {
	np.RLock()
	defer np.RUnlock()

	if len(np.nodes) < 1 {
		return nil, nil
	}

	var d int
	if filterLocal {
		d = 1
	}

	nodes := make([]base.Node, len(np.nodes)-d)
	channels := make([]Channel, len(np.nodes)-d)

	addr := np.local.Address().String()
	var i int
	for k := range np.nodes {
		if filterLocal && k == addr {
			continue
		}

		nodes[i] = np.nodes[k]
		channels[i] = np.chs[k]

		i++
	}

	return nodes, channels
}
