package node

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	BaseV0Type   = hint.Type("base-node")
	BaseV0Hint   = hint.NewHint(BaseV0Type, "v0.0.1")
	BaseV0Hinter = BaseV0{BaseHinter: hint.NewBaseHinter(BaseV0Hint)}
)

type BaseV0 struct {
	hint.BaseHinter
	address   base.Address
	publickey key.Publickey
}

func NewBaseV0(address base.Address, publickey key.Publickey) BaseV0 {
	return BaseV0{BaseHinter: hint.NewBaseHinter(BaseV0Hint), address: address, publickey: publickey}
}

func (bn BaseV0) String() string {
	return bn.address.String()
}

func (bn BaseV0) IsValid([]byte) error {
	return isvalid.Check(nil, false,
		bn.BaseHinter,
		bn.address,
		bn.publickey,
	)
}

func (bn BaseV0) Address() base.Address {
	return bn.address
}

func (bn BaseV0) Publickey() key.Publickey {
	return bn.publickey
}
