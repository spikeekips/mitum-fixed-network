// +build test

package channetwork

import "github.com/spikeekips/mitum/network"

func (ch *Channel) GetBlockDataMapsHandler() network.BlockDataMapsHandler {
	return ch.getBlockDataMaps
}

func (ch *Channel) GetBlockDataHandler() network.BlockDataHandler {
	return ch.getBlockData
}
