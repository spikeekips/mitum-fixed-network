package main

import (
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type Node struct {
	homeState *isaac.HomeState
}

func (n *Node) Home() node.Home {
	return n.homeState.Home()
}

func NewNode(i uint, config *NodeConfig) (*Node, error) {
	//home := node.NewRandomHome()
	n := &Node{}

	return n, nil
}
