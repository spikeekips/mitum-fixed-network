// +build test

package isaac

import (
	"fmt"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/network"
	channetwork "github.com/spikeekips/mitum/network/gochan"
)

func RandomLocalNode(name string, ch network.Channel) *LocalNode {
	pk, _ := key.NewBTCPrivatekey()

	ln := NewLocalNode(
		base.MustStringAddress(fmt.Sprintf("n-%s", name)),
		pk,
	)

	if ch == nil {
		ch = channetwork.NewChannel(0)
	}

	return ln.SetChannel(ch)
}
