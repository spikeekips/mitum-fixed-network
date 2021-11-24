package channetwork

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
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
	connInfo           network.ConnInfo
	recvChan           chan network.PassthroughedSeal
	getSealHandler     network.GetSealsHandler
	getProposalHandler network.GetProposalHandler
	getState           network.GetStateHandler
	nodeInfo           network.NodeInfoHandler
	getBlockDataMaps   network.BlockDataMapsHandler
	getBlockData       network.BlockDataHandler
	startHandover      network.StartHandoverHandler
	pingHandover       network.PingHandoverHandler
	endHandover        network.EndHandoverHandler
}

func NewChannel(bufsize uint, connInfo network.ConnInfo) *Channel {
	return &Channel{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "chan-network-channel")
		}),
		connInfo: connInfo,
		recvChan: make(chan network.PassthroughedSeal, bufsize),
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
		return nil, errors.Errorf("getSealHandler is missing")
	}

	return ch.getSealHandler(h)
}

func (ch *Channel) SendSeal(_ context.Context, ci network.ConnInfo, sl seal.Seal) error {
	ch.recvChan <- network.NewPassthroughedSealFromConnInfo(sl, ci)

	return nil
}

func (ch *Channel) ReceiveSeal() <-chan network.PassthroughedSeal {
	return ch.recvChan
}

func (ch *Channel) SetGetSealHandler(f network.GetSealsHandler) {
	ch.getSealHandler = f
}

func (ch *Channel) Proposal(_ context.Context, h valuehash.Hash) (base.Proposal, error) {
	if ch.getProposalHandler == nil {
		return nil, errors.Errorf("getProposalHandler is missing")
	}

	return ch.getProposalHandler(h)
}

func (ch *Channel) SetGetProposalHandler(f network.GetProposalHandler) {
	ch.getProposalHandler = f
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
		return nil, errors.Errorf("not supported")
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
		return nil, errors.Errorf("not supported")
	}

	return network.FetchBlockDataThruChannel(ch.getBlockData, item)
}

func (ch *Channel) SetBlockDataHandler(f network.BlockDataHandler) {
	ch.getBlockData = f
}

func (ch *Channel) StartHandover(_ context.Context, sl network.StartHandoverSeal) (bool, error) {
	if ch.startHandover == nil {
		return false, errors.Errorf("not supported")
	}

	return ch.startHandover(sl)
}

func (ch *Channel) SetStartHandover(f network.StartHandoverHandler) {
	ch.startHandover = f
}

func (ch *Channel) PingHandover(_ context.Context, sl network.PingHandoverSeal) (bool, error) {
	if ch.pingHandover == nil {
		return false, errors.Errorf("not supported")
	}

	return ch.pingHandover(sl)
}

func (ch *Channel) SetPingHandover(f network.PingHandoverHandler) {
	ch.pingHandover = f
}

func (ch *Channel) EndHandover(_ context.Context, sl network.EndHandoverSeal) (bool, error) {
	if ch.endHandover == nil {
		return false, errors.Errorf("not supported")
	}

	return ch.endHandover(sl)
}

func (ch *Channel) SetEndHandover(f network.EndHandoverHandler) {
	ch.endHandover = f
}
