// +build test

package mitum

import (
	"fmt"

	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/network"
)

func RandomLocalNode(name string, channel network.Channel) *LocalNode {
	pk, _ := key.NewBTCPrivatekey()

	ln := NewLocalNode(
		NewShortAddress(fmt.Sprintf("n-%s", name)),
		pk,
	)

	if channel == nil {
		channel = network.NewChanChannel(0)
	}

	return ln.SetChannel(channel)
}
