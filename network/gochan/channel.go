package channetwork

import (
	"context"
	"io"

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
	connInfo         network.ConnInfo
	recvChan         chan seal.Seal
	getSealHandler   network.GetSealsHandler
	getState         network.GetStateHandler
	nodeInfo         network.NodeInfoHandler
	getBlockDataMaps network.BlockDataMapsHandler
	getBlockData     network.BlockDataHandler
}

func NewChannel(bufsize uint, connInfo network.ConnInfo) *Channel {
	return &Channel{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "chan-network")
		}),
		connInfo: connInfo,
		recvChan: make(chan seal.Seal, bufsize),
	}
}

func (*Channel) Initialize() error {
	return nil
}

func (ch *Channel) ConnInfo() network.ConnInfo {
	return ch.connInfo
}

func (ch *Channel) Seals(_ context.Context, h []valuehash.Hash) ([]seal.Seal, error) {
	if ch.getSealHandler == nil {
		return nil, xerrors.Errorf("getSealHandler is missing")
	}

	return ch.getSealHandler(h)
}

func (ch *Channel) SendSeal(_ context.Context, sl seal.Seal) error {
	ch.recvChan <- sl

	return nil
}

func (ch *Channel) ReceiveSeal() <-chan seal.Seal {
	return ch.recvChan
}

func (ch *Channel) SetGetSealHandler(f network.GetSealsHandler) {
	ch.getSealHandler = f
}

func (ch *Channel) State(_ context.Context, key string) (state.State, bool, error) {
	return ch.getState(key)
}

func (ch *Channel) NodeInfo(_ context.Context) (network.NodeInfo, error) {
	if ch.nodeInfo == nil {
		return nil, nil
	}

	return ch.nodeInfo()
}

func (ch *Channel) SetNodeInfoHandler(f network.NodeInfoHandler) {
	ch.nodeInfo = f
}

func (ch *Channel) BlockDataMaps(_ context.Context, hs []base.Height) ([]block.BlockDataMap, error) {
	if ch.getBlockDataMaps == nil {
		return nil, xerrors.Errorf("not supported")
	}

	bds, err := ch.getBlockDataMaps(hs)
	if err != nil {
		return nil, err
	}
	for i := range bds {
		if err := bds[i].IsValid(nil); err != nil {
			return nil, err
		}
	}

	return bds, nil
}

func (ch *Channel) SetBlockDataMapsHandler(f network.BlockDataMapsHandler) {
	ch.getBlockDataMaps = f
}

func (ch *Channel) BlockData(_ context.Context, item block.BlockDataMapItem) (io.ReadCloser, error) {
	if ch.getBlockData == nil {
		return nil, xerrors.Errorf("not supported")
	}

	return network.FetchBlockDataThruChannel(ch.getBlockData, item)
}

func (ch *Channel) SetBlockDataHandler(f network.BlockDataHandler) {
	ch.getBlockData = f
}
