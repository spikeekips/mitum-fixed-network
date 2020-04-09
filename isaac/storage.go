package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/state"
	"github.com/spikeekips/mitum/util/encoder"
)

type Storage interface {
	Encoder() encoder.Encoder
	Encoders() *encoder.Encoders

	OpenBlockStorage(Block) (BlockStorage, error)
	SyncerStorage() SyncerStorage

	LastBlock() (Block, error)
	Block(valuehash.Hash) (Block, error)
	BlockByHeight(base.Height) (Block, error)
	Manifest(valuehash.Hash) (Manifest, error)
	ManifestByHeight(base.Height) (Manifest, error)

	NewINITVoteproof(base.Voteproof) error
	LastINITVoteproof() (base.Voteproof, error)
	LastINITVoteproofOfHeight(base.Height) (base.Voteproof, error)
	NewACCEPTVoteproof(base.Voteproof) error
	LastACCEPTVoteproof() (base.Voteproof, error)
	LastACCEPTVoteproofOfHeight(base.Height) (base.Voteproof, error)
	Voteproofs(func(base.Voteproof) (bool, error), bool /* sort */) error

	NewSeals([]seal.Seal) error
	Seal(valuehash.Hash) (seal.Seal, error)
	Seals(func(valuehash.Hash, seal.Seal) (bool, error), bool /* sort */, bool /* load Seal? */) error

	NewProposal(Proposal) error
	Proposal(base.Height, base.Round) (Proposal, error)
	Proposals(func(Proposal) (bool, error), bool /* sort */) error

	State(key string) (state.State, bool, error)
	NewState(state.State) error

	HasOperation(valuehash.Hash) (bool, error)

	// NOTE StagedOperationSeals returns the new(staged) operation.Seal by incoming order.
	StagedOperationSeals(func(operation.Seal) (bool, error), bool /* sort */) error
	UnstagedOperationSeals([]valuehash.Hash /* seal.Hash()s */) error
}

type BlockStorage interface {
	Block() Block
	SetBlock(Block) error
	// NOTE UnstageOperationSeals cleans staged operation.Seals
	UnstageOperationSeals([]valuehash.Hash) error
	Commit() error
}
