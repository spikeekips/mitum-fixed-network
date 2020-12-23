// +build test

package network

import (
	"fmt"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
)

func RandomLocalNode(name string, ch Channel) *LocalNode {
	pk, _ := key.NewBTCPrivatekey()

	ln := NewLocalNode(
		base.MustStringAddress(fmt.Sprintf("n-%s", name)),
		pk,
		"",
	)

	return ln.SetChannel(ch)
}
