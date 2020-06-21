package channetwork

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/logging"
)

type NetworkChanChannel struct {
	*logging.Logging
	recvChan       chan seal.Seal
	getSealHandler network.GetSealsHandler
	getManifests   network.GetManifestsHandler
	getBlocks      network.GetBlocksHandler
	nodeInfo       network.NodeInfoHandler
}

func NewNetworkChanChannel(bufsize uint) *NetworkChanChannel {
	return &NetworkChanChannel{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "chan-network")
		}),
		recvChan: make(chan seal.Seal, bufsize),
	}
}

func (gs *NetworkChanChannel) Initialize() error {
	return nil
}

func (gs *NetworkChanChannel) URL() string {
	return "gochan://"
}

func (gs *NetworkChanChannel) Seals(h []valuehash.Hash) ([]seal.Seal, error) {
	if gs.getSealHandler == nil {
		return nil, xerrors.Errorf("getSealHandler is missing")
	}

	return gs.getSealHandler(h)
}

func (gs *NetworkChanChannel) SendSeal(sl seal.Seal) error {
	gs.recvChan <- sl

	return nil
}

func (gs *NetworkChanChannel) ReceiveSeal() <-chan seal.Seal {
	return gs.recvChan
}

func (gs *NetworkChanChannel) SetGetSealHandler(f network.GetSealsHandler) {
	gs.getSealHandler = f
}

func (gs *NetworkChanChannel) Manifests(hs []base.Height) ([]block.Manifest, error) {
	return gs.getManifests(hs)
}

func (gs *NetworkChanChannel) SetGetManifestsHandler(f network.GetManifestsHandler) {
	gs.getManifests = f
}

func (gs *NetworkChanChannel) Blocks(hs []base.Height) ([]block.Block, error) {
	return gs.getBlocks(hs)
}

func (gs *NetworkChanChannel) NodeInfo() (network.NodeInfo, error) {
	if gs.nodeInfo == nil {
		return nil, nil
	}

	return gs.nodeInfo()
}

func (gs *NetworkChanChannel) SetNodeInfoHandler(f network.NodeInfoHandler) {
	gs.nodeInfo = f
}

func (gs *NetworkChanChannel) SetGetBlocksHandler(f network.GetBlocksHandler) {
	gs.getBlocks = f
}
