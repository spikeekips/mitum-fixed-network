package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
)

type (
	GetSealsHandler     func([]valuehash.Hash) ([]seal.Seal, error)
	NewSealHandler      func(seal.Seal) error
	GetManifestsHandler func([]base.Height) ([]block.Manifest, error)
	GetBlocksHandler    func([]base.Height) ([]block.Block, error)
)

// TODO GetXXX should have limit

type Server interface {
	util.Daemon
	SetGetSealsHandler(GetSealsHandler)
	SetNewSealHandler(NewSealHandler)
	SetGetManifests(GetManifestsHandler)
	SetGetBlocks(GetBlocksHandler)
}

type Response interface {
	util.Byter
	OK() bool
}

type NetworkChannel interface {
	Seals([]valuehash.Hash) ([]seal.Seal, error)
	SendSeal(seal.Seal) error
	Manifests([]base.Height) ([]block.Manifest, error)
	Blocks([]base.Height) ([]block.Block, error)
}
