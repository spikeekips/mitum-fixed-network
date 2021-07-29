// +build test

package isaac

import (
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
)

type Local struct {
	database  storage.Database
	blockData blockdata.BlockData
	node      *node.Local
	ch        network.Channel
	policy    *LocalPolicy
	nodes     *network.Nodepool
	networkID []byte
}

func NewLocal(
	st storage.Database,
	blockData blockdata.BlockData,
	node *node.Local,
	ch network.Channel,
	networkID []byte,
) (*Local, error) {
	return &Local{
		database:  st,
		blockData: blockData,
		node:      node,
		ch:        ch,
		nodes:     network.NewNodepool(node, ch),
		networkID: networkID,
	}, nil
}

func (ls *Local) Initialize() error {
	lp := NewLocalPolicy(ls.networkID)

	ls.policy = lp

	return nil
}

func (ls *Local) Database() storage.Database {
	return ls.database
}

func (ls *Local) SetDatabase(st storage.Database) *Local {
	ls.database = st

	return ls
}

func (ls *Local) BlockData() blockdata.BlockData {
	return ls.blockData
}

func (ls *Local) Node() *node.Local {
	return ls.node
}

func (ls *Local) Channel() network.Channel {
	return ls.ch
}

func (ls *Local) SetChannel(ch network.Channel) *Local {
	ls.ch = ch

	return ls
}

func (ls *Local) Policy() *LocalPolicy {
	return ls.policy
}

func (ls *Local) Nodes() *network.Nodepool {
	return ls.nodes
}
