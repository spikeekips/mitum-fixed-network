package isaac

type SyncerStorage interface {
	Manifest(Height) (Manifest, error)
	Manifests([]Height) ([]Manifest, error)
	SetManifests([]Manifest) error
	HasBlock(Height) (bool, error)
	Block(Height) (Block, error)
	Blocks([]Height) ([]Block, error)
	SetBlocks([]Block) error
	Commit() error
	Close() error
}
