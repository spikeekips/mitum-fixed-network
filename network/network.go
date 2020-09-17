package network

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type (
	GetSealsHandler     func([]valuehash.Hash) ([]seal.Seal, error)
	HasSealHandler      func(valuehash.Hash) (bool, error)
	NewSealHandler      func(seal.Seal) error
	GetManifestsHandler func([]base.Height) ([]block.Manifest, error)
	GetBlocksHandler    func([]base.Height) ([]block.Block, error)
	GetStateHandler     func(string) (state.State, bool, error)
	NodeInfoHandler     func() (NodeInfo, error)
)

// TODO GetXXX should have limit

type Server interface {
	util.Daemon
	util.Initializer
	SetHasSealHandler(HasSealHandler)
	SetGetSealsHandler(GetSealsHandler)
	SetNewSealHandler(NewSealHandler)
	SetGetManifestsHandler(GetManifestsHandler)
	SetGetBlocksHandler(GetBlocksHandler)
	SetNodeInfoHandler(NodeInfoHandler)
}

type Response interface {
	util.Byter
	OK() bool
}

type Channel interface {
	util.Initializer
	URL() string
	Seals([]valuehash.Hash) ([]seal.Seal, error)
	SendSeal(seal.Seal) error
	Manifests([]base.Height) ([]block.Manifest, error)
	Blocks([]base.Height) ([]block.Block, error)
	NodeInfo() (NodeInfo, error)
}
