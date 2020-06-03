package storage

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
)

type SyncerStorage interface {
	Manifest(base.Height) (block.Manifest, bool, error)
	Manifests([]base.Height) ([]block.Manifest, error)
	SetManifests([]block.Manifest) error
	HasBlock(base.Height) (bool, error)
	Block(base.Height) (block.Block, bool, error)
	Blocks([]base.Height) ([]block.Block, error)
	SetBlocks([]block.Block) error
	Commit() error
	Close() error
}
