package network

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

type DummyChannel struct {
	connInfo                   ConnInfo
	getStagedOperationsHandler GetStagedOperationsHandler
	newSealHandler             NewSealHandler
	getProposalHandler         GetProposalHandler
	getStateHandler            GetStateHandler
	nodeInfoHandler            NodeInfoHandler
	blockDataMapsHandler       BlockDataMapsHandler
	blockDataHandler           BlockDataHandler
	startHandover              StartHandoverHandler
	pingHandover               PingHandoverHandler
	endHandover                EndHandoverHandler
}

func NewDummyChannel(connInfo ConnInfo) *DummyChannel {
	return &DummyChannel{connInfo: connInfo}
}

func (*DummyChannel) Initialize() error {
	return nil
}

func (ch *DummyChannel) ConnInfo() ConnInfo {
	return ch.connInfo
}

func (ch *DummyChannel) SendSeal(_ context.Context, _ ConnInfo, sl seal.Seal) error {
	if ch.newSealHandler == nil {
		return ch.notSupported()
	}

	return ch.newSealHandler(sl)
}

func (ch *DummyChannel) SetNewSealHandler(f NewSealHandler) {
	ch.newSealHandler = f
}

func (ch *DummyChannel) StagedOperations(_ context.Context, h []valuehash.Hash) ([]operation.Operation, error) {
	if ch.getStagedOperationsHandler == nil {
		return nil, ch.notSupported()
	}

	return ch.getStagedOperationsHandler(h)
}

func (ch *DummyChannel) Proposal(_ context.Context, h valuehash.Hash) (base.Proposal, error) {
	if ch.getProposalHandler == nil {
		return nil, ch.notSupported()
	}

	return ch.getProposalHandler(h)
}

func (ch *DummyChannel) SetGetStagedOperationsHandler(f GetStagedOperationsHandler) {
	ch.getStagedOperationsHandler = f
}

func (ch *DummyChannel) State(_ context.Context, key string) (state.State, bool, error) {
	if ch.getStateHandler == nil {
		return nil, false, ch.notSupported()
	}

	return ch.getStateHandler(key)
}

func (ch *DummyChannel) SetGetStateHandler(f GetStateHandler) {
	ch.getStateHandler = f
}

func (ch *DummyChannel) NodeInfo(_ context.Context) (NodeInfo, error) {
	if ch.nodeInfoHandler == nil {
		return nil, ch.notSupported()
	}

	return ch.nodeInfoHandler()
}

func (ch *DummyChannel) SetNodeInfoHandler(f NodeInfoHandler) {
	ch.nodeInfoHandler = f
}

func (ch *DummyChannel) BlockDataMaps(_ context.Context, heights []base.Height) ([]block.BlockDataMap, error) {
	if ch.blockDataMapsHandler == nil {
		return nil, ch.notSupported()
	}

	return ch.blockDataMapsHandler(heights)
}

func (ch *DummyChannel) SetBlockDataMapsHandler(f BlockDataMapsHandler) {
	ch.blockDataMapsHandler = f
}

func (ch *DummyChannel) BlockData(_ context.Context, item block.BlockDataMapItem) (io.ReadCloser, error) {
	if ch.blockDataHandler == nil {
		return nil, ch.notSupported()
	}

	return FetchBlockDataThruChannel(ch.blockDataHandler, item)
}

func (ch *DummyChannel) SetBlockDataHandler(f BlockDataHandler) {
	ch.blockDataHandler = f
}

func (ch *DummyChannel) StartHandover(_ context.Context, sl StartHandoverSeal) (bool, error) {
	if ch.startHandover == nil {
		return false, ch.notSupported()
	}

	return ch.startHandover(sl)
}

func (ch *DummyChannel) SetStartHandover(f StartHandoverHandler) {
	ch.startHandover = f
}

func (ch *DummyChannel) PingHandover(_ context.Context, sl PingHandoverSeal) (bool, error) {
	if ch.pingHandover == nil {
		return false, ch.notSupported()
	}

	return ch.pingHandover(sl)
}

func (ch *DummyChannel) SetPingHandover(f PingHandoverHandler) {
	ch.pingHandover = f
}

func (ch *DummyChannel) EndHandover(_ context.Context, sl EndHandoverSeal) (bool, error) {
	if ch.endHandover == nil {
		return false, ch.notSupported()
	}

	return ch.endHandover(sl)
}

func (ch *DummyChannel) SetEndHandover(f EndHandoverHandler) {
	ch.endHandover = f
}

func (*DummyChannel) notSupported() error {
	return errors.Errorf("not supported")
}
