package network

import (
	"context"

	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

type Network interface {
	Broadcast(seal.Seal) error
	Request(context.Context, node.Address, seal.Seal) (seal.Seal, error)
	RequestAll(context.Context, seal.Seal) (map[node.Address]seal.Seal, error)
}
