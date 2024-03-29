package channetwork

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Channel struct {
	*logging.Logging
	connInfo                   network.ConnInfo
	recvChan                   chan network.PassthroughedSeal
	getStagedOperationsHandler network.GetStagedOperationsHandler
	getProposalHandler         network.GetProposalHandler
	getState                   network.GetStateHandler
	nodeInfo                   network.NodeInfoHandler
	getBlockdataMaps           network.BlockdataMapsHandler
	getBlockdata               network.BlockdataHandler
	startHandover              network.StartHandoverHandler
	pingHandover               network.PingHandoverHandler
	endHandover                network.EndHandoverHandler
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

func (ch *Channel) StagedOperations(_ context.Context, h []valuehash.Hash) ([]operation.Operation, error) {
	if ch.getStagedOperationsHandler == nil {
		return nil, errors.Errorf("getStagedOperationsHandler is missing")
	}

	return ch.getStagedOperationsHandler(h)
}

func (ch *Channel) SendSeal(_ context.Context, ci network.ConnInfo, sl seal.Seal) error {
	ch.recvChan <- network.NewPassthroughedSealFromConnInfo(sl, ci)

	return nil
}

func (ch *Channel) ReceiveSeal() <-chan network.PassthroughedSeal {
	return ch.recvChan
}

func (ch *Channel) SetGetStagedOperationsHandler(f network.GetStagedOperationsHandler) {
	ch.getStagedOperationsHandler = f
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

func (ch *Channel) BlockdataMaps(_ context.Context, hs []base.Height) ([]block.BlockdataMap, error) {
	if ch.getBlockdataMaps == nil {
		return nil, errors.Errorf("not supported")
	}

	bds, err := ch.getBlockdataMaps(hs)
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

func (ch *Channel) SetBlockdataMapsHandler(f network.BlockdataMapsHandler) {
	ch.getBlockdataMaps = f
}

func (ch *Channel) Blockdata(_ context.Context, item block.BlockdataMapItem) (io.ReadCloser, error) {
	if ch.getBlockdata == nil {
		return nil, errors.Errorf("not supported")
	}

	return network.FetchBlockdataThruChannel(ch.getBlockdata, item)
}

func (ch *Channel) SetBlockdataHandler(f network.BlockdataHandler) {
	ch.getBlockdata = f
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
