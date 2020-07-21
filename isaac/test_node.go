// +build test

package isaac

import (
	"fmt"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/network"
	channetwork "github.com/spikeekips/mitum/network/gochan"
)

func RandomLocalNode(name string, channel network.NetworkChannel) *LocalNode {
	pk, _ := key.NewBTCPrivatekey()

	ln := NewLocalNode(
		base.MustStringAddress(fmt.Sprintf("n-%s", name)),
		pk,
	)

	if channel == nil {
		channel = channetwork.NewNetworkChanChannel(0)
	}

	return ln.SetChannel(channel)
}
