package block

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
)

type Manifest interface {
	isvalid.IsValider
	hint.Hinter
	util.Byter
	valuehash.Hasher
	logging.LogHintedMarshaler
	PreviousBlock() valuehash.Hash
	Height() base.Height
	Round() base.Round
	Proposal() valuehash.Hash
	OperationsHash() valuehash.Hash
	StatesHash() valuehash.Hash
	CreatedAt() time.Time
}

type BlockConsensusInfo interface {
	isvalid.IsValider
	hint.Hinter
	INITVoteproof() base.Voteproof
	ACCEPTVoteproof() base.Voteproof
	SuffrageInfo() SuffrageInfo
}

type Block interface {
	Manifest
	BlockConsensusInfo
	Manifest() Manifest
	ConsensusInfo() BlockConsensusInfo
	Operations() *tree.AVLTree
	States() *tree.AVLTree
}

type BlockUpdater interface {
	Block
	SetINITVoteproof(base.Voteproof) BlockUpdater
	SetACCEPTVoteproof(base.Voteproof) BlockUpdater
	SetOperations(*tree.AVLTree) BlockUpdater
	SetStates(*tree.AVLTree) BlockUpdater
}

type SuffrageInfo interface {
	isvalid.IsValider
	hint.Hinter
	Proposer() base.Address
	Nodes() []base.Node
}
