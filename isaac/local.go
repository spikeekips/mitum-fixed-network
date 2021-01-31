package isaac

import (
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
)

type Local struct {
	storage   storage.Storage
	blockfs   *storage.BlockFS
	node      *network.LocalNode
	policy    *LocalPolicy
	nodes     *network.Nodepool
	networkID []byte
}

func NewLocal(
	st storage.Storage,
	blockfs *storage.BlockFS,
	node *network.LocalNode,
	networkID []byte,
) (*Local, error) {
	return &Local{
		storage:   st,
		blockfs:   blockfs,
		node:      node,
		nodes:     network.NewNodepool(node),
		networkID: networkID,
	}, nil
}

func (ls *Local) Initialize() error {
	lp := NewLocalPolicy(ls.networkID)

	ls.policy = lp

	return nil
}

func (ls *Local) Storage() storage.Storage {
	return ls.storage
}

func (ls *Local) BlockFS() *storage.BlockFS {
	return ls.blockfs
}

func (ls *Local) SetStorage(st storage.Storage) *Local {
	ls.storage = st

	return ls
}

func (ls *Local) Node() *network.LocalNode {
	return ls.node
}

func (ls *Local) Policy() *LocalPolicy {
	return ls.policy
}

func (ls *Local) Nodes() *network.Nodepool {
	return ls.nodes
}
