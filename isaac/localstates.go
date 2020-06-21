package isaac

import (
	"github.com/spikeekips/mitum/storage"
)

type Localstate struct {
	storage   storage.Storage
	node      *LocalNode
	policy    *LocalPolicy
	nodes     *NodesState
	networkID []byte
}

func NewLocalstate(st storage.Storage, node *LocalNode, networkID []byte) (*Localstate, error) {
	return &Localstate{
		storage:   st,
		node:      node,
		nodes:     NewNodesState(node, nil),
		networkID: networkID,
	}, nil
}

func (ls *Localstate) Initialize() error {
	if p, err := NewLocalPolicy(ls.storage, ls.networkID); err != nil {
		return err
	} else {
		ls.policy = p
	}

	return nil
}

func (ls *Localstate) Storage() storage.Storage {
	return ls.storage
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
