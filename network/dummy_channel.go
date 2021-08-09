package network

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
)

type DummyChannel struct {
	connInfo             ConnInfo
	getSealsHandler      GetSealsHandler
	newSealHandler       NewSealHandler
	getStateHandler      GetStateHandler
	nodeInfoHandler      NodeInfoHandler
	blockDataMapsHandler BlockDataMapsHandler
	blockDataHandler     BlockDataHandler
}

func NewDummyChannel(connInfo ConnInfo) *DummyChannel {
	return &DummyChannel{connInfo: connInfo}
}

func (*DummyChannel) Initialize() error {
	return nil
}

func (lc *DummyChannel) ConnInfo() ConnInfo {
	return lc.connInfo
}

func (lc *DummyChannel) SendSeal(_ context.Context, sl seal.Seal) error {
	if lc.newSealHandler == nil {
		return lc.notSupported()
	}

	return lc.newSealHandler(sl)
}

func (lc *DummyChannel) SetNewSealHandler(f NewSealHandler) {
	lc.newSealHandler = f
}

func (lc *DummyChannel) Seals(_ context.Context, h []valuehash.Hash) ([]seal.Seal, error) {
	if lc.getSealsHandler == nil {
		return nil, lc.notSupported()
	}

	return lc.getSealsHandler(h)
}

func (lc *DummyChannel) SetGetSealsHandler(f GetSealsHandler) {
	lc.getSealsHandler = f
}

func (lc *DummyChannel) State(_ context.Context, key string) (state.State, bool, error) {
	if lc.getStateHandler == nil {
		return nil, false, lc.notSupported()
	}

	return lc.getStateHandler(key)
}

func (lc *DummyChannel) SetGetStateHandler(f GetStateHandler) {
	lc.getStateHandler = f
}

func (lc *DummyChannel) NodeInfo(_ context.Context) (NodeInfo, error) {
	if lc.nodeInfoHandler == nil {
		return nil, lc.notSupported()
	}

	return lc.nodeInfoHandler()
}

func (lc *DummyChannel) SetNodeInfoHandler(f NodeInfoHandler) {
	lc.nodeInfoHandler = f
}

func (lc *DummyChannel) BlockDataMaps(_ context.Context, heights []base.Height) ([]block.BlockDataMap, error) {
	if lc.blockDataMapsHandler == nil {
		return nil, lc.notSupported()
	}

	return lc.blockDataMapsHandler(heights)
}

func (lc *DummyChannel) SetBlockDataMapsHandler(f BlockDataMapsHandler) {
	lc.blockDataMapsHandler = f
}

func (lc *DummyChannel) BlockData(_ context.Context, item block.BlockDataMapItem) (io.ReadCloser, error) {
	if lc.blockDataHandler == nil {
		return nil, lc.notSupported()
	}

	return FetchBlockDataThruChannel(lc.blockDataHandler, item)
}

func (lc *DummyChannel) SetBlockDataHandler(f BlockDataHandler) {
	lc.blockDataHandler = f
}

func (*DummyChannel) notSupported() error {
	return errors.Errorf("not supported")
}
