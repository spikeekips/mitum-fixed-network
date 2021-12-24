// +build test

package channetwork

import "github.com/spikeekips/mitum/network"

func (ch *Channel) GetBlockdataMapsHandler() network.BlockdataMapsHandler {
	return ch.getBlockdataMaps
}

func (ch *Channel) GetBlockdataHandler() network.BlockdataHandler {
	return ch.getBlockdata
}

func RandomChannel(name string) network.Channel {
	return NewChannel(0, network.NewNilConnInfo(name))
}
