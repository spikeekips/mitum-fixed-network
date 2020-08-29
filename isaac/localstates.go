package isaac

import (
	"github.com/spikeekips/mitum/storage"
)

type Localstate struct {
	storage   storage.Storage
	blockfs   *storage.BlockFS
	node      *LocalNode
	policy    *LocalPolicy
	nodes     *NodesState
	networkID []byte
}

func NewLocalstate(
	st storage.Storage,
	blockfs *storage.BlockFS,
	node *LocalNode,
	networkID []byte,
) (*Localstate, error) {
	return &Localstate{
		storage:   st,
		blockfs:   blockfs,
		node:      node,
		nodes:     NewNodesState(node, nil),
		networkID: networkID,
	}, nil
}

func (ls *Localstate) Initialize() error {
	lp := NewLocalPolicy(ls.networkID)
	if ls.storage != nil {
		if err := lp.Reload(ls.storage); err != nil {
			return err
		}

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

func (ls *Localstate) Storage() storage.Storage {
	return ls.storage
}

func (ls *Localstate) BlockFS() *storage.BlockFS {
	return ls.blockfs
}

func (ls *Localstate) SetStorage(st storage.Storage) *Localstate {
	ls.storage = st

	return ls
}

func (ls *Localstate) Node() *LocalNode {
	return ls.node
}

func (ls *Localstate) Policy() *LocalPolicy {
	return ls.policy
}

func (ls *Localstate) Nodes() *NodesState {
	return ls.nodes
}
