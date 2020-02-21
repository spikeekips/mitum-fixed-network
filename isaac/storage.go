package isaac

import (
	"github.com/spikeekips/avl"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type Storage interface {
	Encoder() encoder.Encoder
	Encoders() *encoder.Encoders

	LastBlock() (Block, error)
	Block(valuehash.Hash) (Block, error)
	BlockByHeight(Height) (Block, error)

	NewINITVoteproof(Voteproof) error
	LastINITVoteproof() (Voteproof, error)
	LastINITVoteproofOfHeight(Height) (Voteproof, error)
	NewACCEPTVoteproof(Voteproof) error
	LastACCEPTVoteproof() (Voteproof, error)
	LastACCEPTVoteproofOfHeight(Height) (Voteproof, error)
	Voteproofs(func(Voteproof) (bool, error), bool /* sort */) error

	NewSeal(seal.Seal) error
	Seal(valuehash.Hash) (seal.Seal, error)
	Seals(func(seal.Seal) (bool, error), bool /* sort */) error

	NewProposal(Proposal) error
	Proposal(Height, Round) (Proposal, error)
	Proposals(func(Proposal) (bool, error), bool /* sort */) error

	OpenBlockStorage(Block) (BlockStorage, error)
}

type BlockStorage interface {
	Block() Block
	SetOperations(*avl.Tree) error
	SetStates(*avl.Tree) error
	Commit() error
}
