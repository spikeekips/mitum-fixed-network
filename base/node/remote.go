package node

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
)

type Remote struct {
	BaseV0
}

func NewRemote(address base.Address, publickey key.Publickey) *Remote {
	return &Remote{
		BaseV0: NewBaseV0(address, publickey),
	}
}
