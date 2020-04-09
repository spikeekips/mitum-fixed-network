// +build test

package base

import (
	"fmt"

	"github.com/spikeekips/mitum/base/key"
)

type DummyNode struct {
	address    Address
	publickey  key.Publickey
	privatekey key.Privatekey
}

func NewDummyNode(address Address, privatekey key.Privatekey) *DummyNode {
	return &DummyNode{
		address:    address,
		publickey:  privatekey.Publickey(),
		privatekey: privatekey,
	}
}

func (ln *DummyNode) Address() Address {
	return ln.address
}

func (ln *DummyNode) Publickey() key.Publickey {
	return ln.publickey
}

func (ln *DummyNode) Privatekey() key.Privatekey {
	return ln.privatekey
}

func RandomNode(name string) *DummyNode {
	pk, _ := key.NewBTCPrivatekey()

	return NewDummyNode(
		NewShortAddress(fmt.Sprintf("n-%s", name)),
		pk,
	)
}
