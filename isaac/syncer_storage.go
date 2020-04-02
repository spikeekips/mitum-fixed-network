package isaac

type SyncerStorage interface {
	Manifest(Height) (BlockManifest, error)
	Manifests([]Height) ([]BlockManifest, error)
	SetManifests([]BlockManifest) error
	HasBlock(Height) (bool, error)
	Block(Height) (Block, error)
	Blocks([]Height) ([]Block, error)
	SetBlocks([]Block) error
	Commit() error
	Close() error
}
