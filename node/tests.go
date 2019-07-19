// +build test

package node

import (
	"crypto/rand"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/keypair"
)

func init() {
	common.SetTestLogger(Log())
}

func NewRandomAddress() Address {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	h, _ := NewAddress(b)
	return h
}

func NewRandomHome() Home {
	pk, _ := keypair.NewStellarPrivateKey()

	return NewHome(NewRandomAddress(), pk)
}

func NewRandomOther() (Other, keypair.PrivateKey) {
	pk, _ := keypair.NewStellarPrivateKey()

	return NewOther(NewRandomAddress(), pk.PublicKey()), pk
}
