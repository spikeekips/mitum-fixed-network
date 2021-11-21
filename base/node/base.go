package node

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	BaseV0Type = hint.Type("base-node")
	BaseV0Hint = hint.NewHint(BaseV0Type, "v0.0.1")
)

type BaseV0 struct {
	address   base.Address
	publickey key.Publickey
}

func NewBaseV0(address base.Address, publickey key.Publickey) BaseV0 {
	return BaseV0{address: address, publickey: publickey}
}

func (bn BaseV0) String() string {
	return bn.address.String()
}

func (BaseV0) Hint() hint.Hint {
	return BaseV0Hint
}

func (bn BaseV0) IsValid([]byte) error {
	return isvalid.Check([]isvalid.IsValider{
		bn.address,
		bn.publickey,
	}, nil, false)
}

func (bn BaseV0) Address() base.Address {
	return bn.address
}

func (bn BaseV0) Publickey() key.Publickey {
	return bn.publickey
}
