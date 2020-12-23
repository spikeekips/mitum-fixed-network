// +build test

package channetwork

import (
	"github.com/spikeekips/mitum/network"
)

func RandomLocalNode(name string) *network.LocalNode {
	n := network.RandomLocalNode(name, nil)
	n.SetChannel(NewChannel(0))

	return n
}
