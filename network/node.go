package network

import "github.com/spikeekips/mitum/base"

type Node interface {
	base.Node
	Channel() NetworkChannel
}
