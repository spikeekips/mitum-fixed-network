// +build test

package channetwork

import "github.com/spikeekips/mitum/network"

func (gs *NetworkChanChannel) GetBlocksHandler() network.GetBlocksHandler {
	return gs.getBlocks
}

func (gs *NetworkChanChannel) GetManifestsHandler() network.GetManifestsHandler {
	return gs.getManifests
}
