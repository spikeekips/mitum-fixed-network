package network

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

type DummyChannel struct {
	url                 string
	getSealsHandler     GetSealsHandler
	newSealHandler      NewSealHandler
	getManifestsHandler GetManifestsHandler
	getBlocksHandler    GetBlocksHandler
	getStateHandler     GetStateHandler
	nodeInfoHandler     NodeInfoHandler
}

func NewDummyChannel(url string) *DummyChannel {
	return &DummyChannel{url: url}
}

func (lc *DummyChannel) Initialize() error {
	return nil
}

func (lc *DummyChannel) URL() string {
	return lc.url
}

func (lc *DummyChannel) SendSeal(sl seal.Seal) error {
	if lc.newSealHandler == nil {
		return lc.unsupported()
	}

	return lc.newSealHandler(sl)
}

func (lc *DummyChannel) SetNewSealHandler(f NewSealHandler) {
	lc.newSealHandler = f
}

func (lc *DummyChannel) Seals(h []valuehash.Hash) ([]seal.Seal, error) {
	if lc.getSealsHandler == nil {
		return nil, lc.unsupported()
	}

	return lc.getSealsHandler(h)
}

func (lc *DummyChannel) SetGetSealsHandler(f GetSealsHandler) {
	lc.getSealsHandler = f
}

func (lc *DummyChannel) Manifests(hs []base.Height) ([]block.Manifest, error) {
	if lc.getManifestsHandler == nil {
		return nil, lc.unsupported()
	}

	return lc.getManifestsHandler(hs)
}

func (lc *DummyChannel) SetGetManifestsHandler(f GetManifestsHandler) {
	lc.getManifestsHandler = f
}

func (lc *DummyChannel) Blocks(hs []base.Height) ([]block.Block, error) {
	if lc.getBlocksHandler == nil {
		return nil, lc.unsupported()
	}

	return lc.getBlocksHandler(hs)
}

func (lc *DummyChannel) SetGetBlocksHandler(f GetBlocksHandler) {
	lc.getBlocksHandler = f
}

func (lc *DummyChannel) State(key string) (state.State, bool, error) {
	if lc.getStateHandler == nil {
		return nil, false, lc.unsupported()
	}

	return lc.getStateHandler(key)
}

func (lc *DummyChannel) SetGetStateHandler(f GetStateHandler) {
	lc.getStateHandler = f
}

func (lc *DummyChannel) NodeInfo() (NodeInfo, error) {
	if lc.nodeInfoHandler == nil {
		return nil, lc.unsupported()
	}

	return lc.nodeInfoHandler()
}

func (lc *DummyChannel) SetNodeInfoHandler(f NodeInfoHandler) {
	lc.nodeInfoHandler = f
}

func (lc *DummyChannel) unsupported() error {
	return xerrors.Errorf("unsupported")
}
