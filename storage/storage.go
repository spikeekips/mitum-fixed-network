package storage

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Database interface {
	util.Initializer
	Encoder() encoder.Encoder
	Encoders() *encoder.Encoders
	Close() error
	Clean() error
	CleanByHeight(base.Height) error
	Copy(Database /* source */) error

	NewSession(block.Block) (DatabaseSession, error)
	NewSyncerSession() (SyncerSession, error)

	LastManifest() (block.Manifest, bool, error)
	Manifest(valuehash.Hash) (block.Manifest, bool, error)
	ManifestByHeight(base.Height) (block.Manifest, bool, error)
	Manifests(bool, bool, int64, func(base.Height, valuehash.Hash, block.Manifest) (bool, error)) error

	NewOperationSeals([]operation.Seal) error
	NewOperations([]operation.Operation) error

	NewProposal(base.Proposal) error
	Proposal(valuehash.Hash /* fact hash */) (base.Proposal, bool, error)
	ProposalByPoint(base.Height, base.Round, base.Address /* proposer address */) (base.Proposal, bool, error)
	Proposals(func(base.Proposal) (bool, error), bool /* sort */) error

	State(key string) (state.State, bool, error)
	LastVoteproof(base.Stage) base.Voteproof
	Voteproof(base.Height, base.Stage) (base.Voteproof, error)

	HasOperationFact(valuehash.Hash) (bool, error)

	// NOTE StagedOperationOperations returns operation.Operation by incoming order.
	StagedOperationsByFact(facts []valuehash.Hash) ([]operation.Operation, error)
	HasStagedOperation(valuehash.Hash) (bool, error)
	StagedOperations(func(operation.Operation) (bool, error), bool /* sort */) error
	UnstagedOperations([]valuehash.Hash /* operation.Fact().Hash()s */) error

	SetInfo(string /* key */, []byte /* value */) error
	Info(string /* key */) ([]byte, bool, error)

	BlockdataMap(base.Height) (block.BlockdataMap, bool, error)
	SetBlockdataMaps([]block.BlockdataMap) error
	LocalBlockdataMapsByHeight(base.Height, func(block.BlockdataMap) (bool, error)) error
}

type DatabaseSession interface {
	Block() block.Block
	SetBlock(context.Context, block.Block) error
	SetACCEPTVoteproof(base.Voteproof) error
	Commit(context.Context, block.BlockdataMap) error
	Close() error
	Cancel() error
}

type SyncerSession interface {
	Manifest(base.Height) (block.Manifest, bool, error)
	SetManifests([]block.Manifest) error
	HasBlock(base.Height) (bool, error)
	SetBlocks([]block.Block, []block.BlockdataMap) error
	Commit() error
	Close() error
	SetSkipLastBlock(bool)
}

type LastBlockSaver interface {
	SaveLastBlock(base.Height) error
}

type StateUpdater interface {
	NewState(state.State) error
}
