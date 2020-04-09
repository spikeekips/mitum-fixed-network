package isaac

import "github.com/spikeekips/mitum/base"

type SyncerStorage interface {
	Manifest(base.Height) (Manifest, error)
	Manifests([]base.Height) ([]Manifest, error)
	SetManifests([]Manifest) error
	HasBlock(base.Height) (bool, error)
	Block(base.Height) (Block, error)
	Blocks([]base.Height) ([]Block, error)
	SetBlocks([]Block) error
	Commit() error
	Close() error
}
