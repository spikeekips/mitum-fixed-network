package discovery

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

type Discovery interface {
	util.Daemon
	LenNodes() int
	Nodes() []NodeConnInfo
	SetNotifyJoin(func(NodeConnInfo)) Discovery
	SetNotifyLeave(func(NodeConnInfo, []NodeConnInfo)) Discovery
	SetNotifyUpdate(func(NodeConnInfo)) Discovery
}

type NodeConnInfo interface {
	network.ConnInfo
	Node() base.Address
}
