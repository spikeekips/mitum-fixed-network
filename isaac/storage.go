package isaac

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/state"
	"github.com/spikeekips/mitum/valuehash"
)

type Storage interface {
	Encoder() encoder.Encoder
	Encoders() *encoder.Encoders

	OpenBlockStorage(Block) (BlockStorage, error)
	SyncerStorage() SyncerStorage

	LastBlock() (Block, error)
	Block(valuehash.Hash) (Block, error)
	BlockByHeight(Height) (Block, error)
	BlockManifest(valuehash.Hash) (BlockManifest, error)
	BlockManifestByHeight(Height) (BlockManifest, error)

	NewINITVoteproof(Voteproof) error
	LastINITVoteproof() (Voteproof, error)
	LastINITVoteproofOfHeight(Height) (Voteproof, error)
	NewACCEPTVoteproof(Voteproof) error
	LastACCEPTVoteproof() (Voteproof, error)
	LastACCEPTVoteproofOfHeight(Height) (Voteproof, error)
	Voteproofs(func(Voteproof) (bool, error), bool /* sort */) error

	NewSeals([]seal.Seal) error
	Seal(valuehash.Hash) (seal.Seal, error)
	Seals(func(valuehash.Hash, seal.Seal) (bool, error), bool /* sort */, bool /* load Seal? */) error

	NewProposal(Proposal) error
	Proposal(Height, Round) (Proposal, error)
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
