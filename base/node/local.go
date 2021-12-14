package node

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
)

type Local struct {
	BaseV0
	privatekey key.Privatekey
}

func NewLocal(address base.Address, privatekey key.Privatekey) Local {
	return Local{
		BaseV0:     NewBaseV0(address, privatekey.Publickey()),
		privatekey: privatekey,
	}
}

func (ln Local) Publickey() key.Publickey {
	return ln.BaseV0.Publickey()
}

func (ln Local) Privatekey() key.Privatekey {
	return ln.privatekey
}
