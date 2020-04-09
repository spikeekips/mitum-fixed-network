// +build test

package isaac

import (
	"fmt"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
)

func RandomLocalNode(name string, channel NetworkChannel) *LocalNode {
	pk, _ := key.NewBTCPrivatekey()

	ln := NewLocalNode(
		base.NewShortAddress(fmt.Sprintf("n-%s", name)),
		pk,
	)

	if channel == nil {
		channel = NewNetworkChanChannel(0)
	}

	return ln.SetChannel(channel)
}
