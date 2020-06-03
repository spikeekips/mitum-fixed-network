package storage

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

type Storage interface {
	Encoder() encoder.Encoder
	Encoders() *encoder.Encoders
	Close() error
	Clean() error
	Copy(Storage /* source */) error

	OpenBlockStorage(block.Block) (BlockStorage, error)
	SyncerStorage() (SyncerStorage, error)

	LastBlock() (block.Block, bool, error)
	LastManifest() (block.Manifest, bool, error)
	Block(valuehash.Hash) (block.Block, bool, error)
	BlockByHeight(base.Height) (block.Block, bool, error)
	// TODO add BlocksByHeight([]base.Height) ([]block.Block, error)
	Manifest(valuehash.Hash) (block.Manifest, bool, error)
	ManifestByHeight(base.Height) (block.Manifest, bool, error)
	// TODO add ManifestsByHeight([]base.Height) ([]block.Manifest, error)
	LastVoteproof(base.Stage) (base.Voteproof, bool, error)

	NewSeals([]seal.Seal) error
	Seal(valuehash.Hash) (seal.Seal, bool, error)
	Seals(func(valuehash.Hash, seal.Seal) (bool, error), bool /* sort */, bool /* load Seal? */) error
	HasSeal(valuehash.Hash) (bool, error)
	// TODO add get Seals by []valuehash.Hash

	NewProposal(ballot.Proposal) error
	Proposal(base.Height, base.Round) (ballot.Proposal, bool, error)
	Proposals(func(ballot.Proposal) (bool, error), bool /* sort */) error

	State(key string) (state.State, bool, error)
	NewState(state.State) error

	HasOperation(valuehash.Hash) (bool, error)

	// NOTE StagedOperationSeals returns the new(staged) operation.Seal by incoming order.
	StagedOperationSeals(func(operation.Seal) (bool, error), bool /* sort */) error
	UnstagedOperationSeals([]valuehash.Hash /* seal.Hash()s */) error
}

type BlockStorage interface {
	Block() block.Block
	SetBlock(block.Block) error
	// NOTE UnstageOperationSeals cleans staged operation.Seals
	UnstageOperationSeals([]valuehash.Hash) error
	Commit() error
}

type LastBlockSaver interface {
	SaveLastBlock(base.Height) error
}
