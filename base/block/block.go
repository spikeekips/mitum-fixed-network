package block

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
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

type ConsensusInfo interface {
	isvalid.IsValider
	hint.Hinter
	INITVoteproof() base.Voteproof
	ACCEPTVoteproof() base.Voteproof
	SuffrageInfo() SuffrageInfo
	Proposal() ballot.Proposal
}

type Block interface {
	Manifest
	Manifest() Manifest
	ConsensusInfo() ConsensusInfo
	OperationsTree() tree.FixedTree
	StatesTree() tree.FixedTree
	States() []state.State
	Operations() []operation.Operation
}

type BlockUpdater interface {
	Block
	SetManifest(Manifest) BlockUpdater
	SetINITVoteproof(base.Voteproof) BlockUpdater
	SetACCEPTVoteproof(base.Voteproof) BlockUpdater
	SetOperationsTree(tree.FixedTree) BlockUpdater
	SetOperations([]operation.Operation) BlockUpdater
	SetStatesTree(tree.FixedTree) BlockUpdater
	SetStates([]state.State) BlockUpdater
	SetProposal(ballot.Proposal) BlockUpdater
	SetSuffrageInfo(SuffrageInfo) BlockUpdater
}

type SuffrageInfo interface {
	isvalid.IsValider
	hint.Hinter
	Proposer() base.Address
	Nodes() []base.Node
}
