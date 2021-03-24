package storage

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Storage interface {
	util.Initializer
	Encoder() encoder.Encoder
	Encoders() *encoder.Encoders
	Close() error
	Clean() error
	CleanByHeight(base.Height) error
	Copy(Storage /* source */) error

	NewStorageSession(block.Block) (StorageSession, error)
	NewSyncerSession() (SyncerSession, error)

	LastManifest() (block.Manifest, bool, error)
	Manifest(valuehash.Hash) (block.Manifest, bool, error)
	ManifestByHeight(base.Height) (block.Manifest, bool, error)

	NewSeals([]seal.Seal) error
	Seal(valuehash.Hash) (seal.Seal, bool, error)
	Seals(func(valuehash.Hash, seal.Seal) (bool, error), bool /* sort */, bool /* load Seal? */) error
	HasSeal(valuehash.Hash) (bool, error)
	SealsByHash([]valuehash.Hash, func(valuehash.Hash, seal.Seal) (bool, error), bool /* load Seal? */) error

	NewProposal(ballot.Proposal) error
	Proposal(base.Height, base.Round, base.Address /* proposer address */) (ballot.Proposal, bool, error)
	Proposals(func(ballot.Proposal) (bool, error), bool /* sort */) error

	State(key string) (state.State, bool, error)
	LastVoteproof(base.Stage) base.Voteproof
	Voteproof(base.Height, base.Stage) (base.Voteproof, error)

	HasOperationFact(valuehash.Hash) (bool, error)

	// NOTE StagedOperationSeals returns operation.Seals by incoming order.
	StagedOperationSeals(func(operation.Seal) (bool, error), bool /* sort */) error
	UnstagedOperationSeals([]valuehash.Hash /* seal.Hash()s */) error

	SetInfo(string /* key */, []byte /* value */) error
	Info(string /* key */) ([]byte, bool, error)

	BlockDataMap(base.Height) (block.BlockDataMap, bool, error)
	LocalBlockDataMapsByHeight(base.Height, func(block.BlockDataMap) (bool, error)) error
}

type StorageSession interface {
	Block() block.Block
	SetBlock(context.Context, block.Block) error
	SetACCEPTVoteproof(base.Voteproof) error
	Commit(context.Context, block.BlockDataMap) error
	Close() error
	Cancel() error
}

type SyncerSession interface {
	Manifest(base.Height) (block.Manifest, bool, error)
	Manifests([]base.Height) ([]block.Manifest, error)
	SetManifests([]block.Manifest) error
	HasBlock(base.Height) (bool, error)
	SetBlocks([]block.Block, []block.BlockDataMap) error
	Commit() error
	Close() error
}

type LastBlockSaver interface {
	SaveLastBlock(base.Height) error
}

type StateUpdater interface {
	NewState(state.State) error
}
