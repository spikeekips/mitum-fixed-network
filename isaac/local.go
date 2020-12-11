package isaac

import (
	"github.com/spikeekips/mitum/storage"
)

type Local struct {
	storage   storage.Storage
	blockfs   *storage.BlockFS
	node      *LocalNode
	policy    *LocalPolicy
	nodes     *NodesPool
	networkID []byte
}

func NewLocal(
	st storage.Storage,
	blockfs *storage.BlockFS,
	node *LocalNode,
	networkID []byte,
) (*Local, error) {
	return &Local{
		storage:   st,
		blockfs:   blockfs,
		node:      node,
		nodes:     NewNodesPool(node),
		networkID: networkID,
	}, nil
}

func (ls *Local) Initialize() error {
	lp := NewLocalPolicy(ls.networkID)
	if ls.blockfs != nil {
		if err := ls.blockfs.Initialize(); err != nil {
			return err
		}
	}

	if ls.storage != nil {
		if m, found, err := ls.storage.LastManifest(); err != nil {
			return err
		} else if found {
			if err := ls.blockfs.SetLast(m.Height()); err != nil {
				return err
			}
		}
	}

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

func (ls *Local) Node() *LocalNode {
	return ls.node
}

func (ls *Local) Policy() *LocalPolicy {
	return ls.policy
}

func (ls *Local) Nodes() *NodesPool {
	return ls.nodes
}
