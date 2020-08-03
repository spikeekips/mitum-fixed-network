package channetwork

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Channel struct {
	*logging.Logging
	recvChan       chan seal.Seal
	getSealHandler network.GetSealsHandler
	getManifests   network.GetManifestsHandler
	getBlocks      network.GetBlocksHandler
	getState       network.GetStateHandler
	nodeInfo       network.NodeInfoHandler
}

func NewChannel(bufsize uint) *Channel {
	return &Channel{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "chan-network")
		}),
		recvChan: make(chan seal.Seal, bufsize),
	}
}

func (ch *Channel) Initialize() error {
	return nil
}

func (ch *Channel) URL() string {
	return "gochan://"
}

func (ch *Channel) Seals(h []valuehash.Hash) ([]seal.Seal, error) {
	if ch.getSealHandler == nil {
		return nil, xerrors.Errorf("getSealHandler is missing")
	}

	return ch.getSealHandler(h)
}

func (ch *Channel) SendSeal(sl seal.Seal) error {
	ch.recvChan <- sl

	return nil
}

func (ch *Channel) ReceiveSeal() <-chan seal.Seal {
	return ch.recvChan
}

func (ch *Channel) SetGetSealHandler(f network.GetSealsHandler) {
	ch.getSealHandler = f
}

func (ch *Channel) Manifests(hs []base.Height) ([]block.Manifest, error) {
	return ch.getManifests(hs)
}

func (ch *Channel) SetGetManifestsHandler(f network.GetManifestsHandler) {
	ch.getManifests = f
}

func (ch *Channel) Blocks(hs []base.Height) ([]block.Block, error) {
	return ch.getBlocks(hs)
}

func (ch *Channel) State(key string) (state.State, bool, error) {
	return ch.getState(key)
}

func (ch *Channel) NodeInfo() (network.NodeInfo, error) {
	if ch.nodeInfo == nil {
		return nil, nil
	}

	return ch.nodeInfo()
}

func (ch *Channel) SetNodeInfoHandler(f network.NodeInfoHandler) {
	ch.nodeInfo = f
}

func (ch *Channel) SetGetBlocksHandler(f network.GetBlocksHandler) {
	ch.getBlocks = f
}
