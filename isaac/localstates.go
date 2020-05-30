package isaac

import (
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
)

type Localstate struct {
	storage storage.Storage
	node    *LocalNode
	policy  *LocalPolicy
	nodes   *NodesState
}

func NewLocalstate(st storage.Storage, node *LocalNode, networkID []byte) (*Localstate, error) {
	var policy *LocalPolicy
	if p, err := NewLocalPolicy(st, networkID); err != nil {
		return nil, err
	} else {
		policy = p
	}

	return &Localstate{
		storage: st,
		node:    node,
		policy:  policy,
		nodes:   NewNodesState(node, nil),
	}, nil
}

func (ls *Localstate) Storage() storage.Storage {
	return ls.storage
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

func (ls *Localstate) Seal(h valuehash.Hash) (seal.Seal, error) {
	if ls.storage != nil {
		return ls.storage.Seal(h)
	}

	return nil, nil
}
