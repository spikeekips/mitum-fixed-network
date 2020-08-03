// +build test

package channetwork

import "github.com/spikeekips/mitum/network"

func (ch *Channel) GetBlocksHandler() network.GetBlocksHandler {
	return ch.getBlocks
}

func (ch *Channel) GetManifestsHandler() network.GetManifestsHandler {
	return ch.getManifests
}
