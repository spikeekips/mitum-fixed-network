package isaac

import (
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type (
	GetSealsHandler     func([]valuehash.Hash) ([]seal.Seal, error)
	NewSealHandler      func(seal.Seal) error
	GetManifestsHandler func([]Height) ([]Manifest, error)
	GetBlocksHandler    func([]Height) ([]Block, error)
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
	Manifests([]Height) ([]Manifest, error)
	Blocks([]Height) ([]Block, error)
}
