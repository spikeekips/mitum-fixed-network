package isaac

import (
	"time"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type BlockManifest interface {
	isvalid.IsValider
	hint.Hinter
	util.Byter
	valuehash.Hasher
	PreviousBlock() valuehash.Hash
	Height() Height
	Round() Round
	Proposal() valuehash.Hash
	Operations() valuehash.Hash
	States() valuehash.Hash
	CreatedAt() time.Time
}

type BlockConsensusInfo interface {
	isvalid.IsValider
	hint.Hinter
	util.Byter
	INITVoteproof() Voteproof
	ACCEPTVoteproof() Voteproof
}

type Block interface {
	BlockManifest
	BlockConsensusInfo
}

type BlockUpdater interface {
	Block
	SetINITVoteproof(Voteproof) BlockUpdater
	SetACCEPTVoteproof(Voteproof) BlockUpdater
}
