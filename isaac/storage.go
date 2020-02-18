package isaac

import (
	"github.com/spikeekips/avl"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type Storage interface {
	LastBlock() (Block, error)
	// BlockByHeight(Height) (Block, error)
	// BlockByHash() (Block, error)
	LastINITVoteProof() (VoteProof, error)
	NewINITVoteProof(VoteProof) error
	LastACCEPTVoteProof() (VoteProof, error)
	NewACCEPTVoteProof(VoteProof) error
	// TODO replace SealStorage
	Seal(valuehash.Hash) (seal.Seal, error)
	NewSeal(seal.Seal) error
	Proposal(Height, Round) (Proposal, error)
	NewProposal(Proposal) error
	OpenBlockStorage(Block) (BlockStorage, error)
}

type BlockStorage interface {
	Block() Block
	SetOperations(*avl.Tree) error
	SetStates(*avl.Tree) error
	Commit() error
}
