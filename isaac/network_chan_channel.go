package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/logging"
)

type NetworkChanChannel struct {
	*logging.Logging
	recvChan       chan seal.Seal
	getSealHandler GetSealsHandler
	getManifests   GetManifestsHandler
	getBlocks      GetBlocksHandler
}

func NewNetworkChanChannel(bufsize uint) *NetworkChanChannel {
	return &NetworkChanChannel{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "chan-network")
		}),
		recvChan: make(chan seal.Seal, bufsize),
	}
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

func (gs *NetworkChanChannel) SetGetSealHandler(f GetSealsHandler) {
	gs.getSealHandler = f
}

func (gs *NetworkChanChannel) Manifests(hs []base.Height) ([]block.Manifest, error) {
	return gs.getManifests(hs)
}

func (gs *NetworkChanChannel) SetGetManifests(f GetManifestsHandler) {
	gs.getManifests = f
}

func (gs *NetworkChanChannel) Blocks(hs []base.Height) ([]block.Block, error) {
	return gs.getBlocks(hs)
}

func (gs *NetworkChanChannel) SetGetBlocks(f GetBlocksHandler) {
	gs.getBlocks = f
}
