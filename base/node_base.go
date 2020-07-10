package base

import (
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	BaseNodeV0Type = hint.MustNewType(0x01, 0x70, "base-node-v0")
	BaseNodeV0Hint = hint.MustHint(BaseNodeV0Type, "0.0.1")
)

type BaseNodeV0 struct {
	address   Address
	publickey key.Publickey
}

func NewBaseNodeV0(address Address, publickey key.Publickey) BaseNodeV0 {
	return BaseNodeV0{address: address, publickey: publickey}
}

func (bn BaseNodeV0) String() string {
	return bn.address.String()
}

func (bn BaseNodeV0) Hint() hint.Hint {
	return BaseNodeV0Hint
}

func (bn BaseNodeV0) IsValid([]byte) error {
	return isvalid.Check([]isvalid.IsValider{
		bn.address,
		bn.publickey,
	}, nil, false)
}

func (bn BaseNodeV0) Bytes() []byte {
	return util.ConcatBytesSlice(bn.address.Bytes(), []byte(bn.publickey.String()))
}

func (bn BaseNodeV0) Address() Address {
	return bn.address
}

func (bn BaseNodeV0) Publickey() key.Publickey {
	return bn.publickey
}
