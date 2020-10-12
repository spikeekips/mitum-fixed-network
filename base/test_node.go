// +build test

package base

import (
	"fmt"

	"github.com/spikeekips/mitum/base/key"
)

type DummyNode struct {
	BaseNodeV0
	privatekey key.Privatekey
}

func NewDummyNode(address Address, privatekey key.Privatekey) *DummyNode {
	return &DummyNode{
		BaseNodeV0: NewBaseNodeV0(address, privatekey.Publickey(), ""),
		privatekey: privatekey,
	}
}

func (ln *DummyNode) Privatekey() key.Privatekey {
	return ln.privatekey
}

func (ln *DummyNode) SetURL(url string) *DummyNode {
	ln.BaseNodeV0.url = url

	return ln
}

func RandomNode(name string) *DummyNode {
	pk, _ := key.NewBTCPrivatekey()

	return NewDummyNode(
		MustStringAddress(fmt.Sprintf("n-%s", name)),
		pk,
	)
}
