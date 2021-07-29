// +build test

package node

import (
	"fmt"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
)

type DummyNode struct {
	BaseV0
	privatekey key.Privatekey
}

func NewDummyNode(address base.Address, privatekey key.Privatekey) *DummyNode {
	return &DummyNode{
		BaseV0:     NewBaseV0(address, privatekey.Publickey()),
		privatekey: privatekey,
	}
}

func (ln *DummyNode) Privatekey() key.Privatekey {
	return ln.privatekey
}

func RandomNode(name string) *DummyNode {
	return NewDummyNode(
		base.MustStringAddress(fmt.Sprintf("n-%s", name)),
		key.MustNewBTCPrivatekey(),
	)
}

func RandomLocal(name string) *Local {
	return NewLocal(
		base.MustStringAddress(fmt.Sprintf("n-%s", name)),
		key.MustNewBTCPrivatekey(),
	)
}
