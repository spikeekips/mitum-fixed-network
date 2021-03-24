// +build test

package isaac

import (
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
)

type Local struct {
	storage   storage.Storage
	blockData blockdata.BlockData
	node      *network.LocalNode
	policy    *LocalPolicy
	nodes     *network.Nodepool
	networkID []byte
}

func NewLocal(
	st storage.Storage,
	blockData blockdata.BlockData,
	node *network.LocalNode,
	networkID []byte,
) (*Local, error) {
	return &Local{
		storage:   st,
		blockData: blockData,
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

func (ls *Local) BlockData() blockdata.BlockData {
	return ls.blockData
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
