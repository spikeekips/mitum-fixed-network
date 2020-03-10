package isaac

import (
	"time"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/valuehash"
)

type Block interface {
	isvalid.IsValider
	hint.Hinter
	Bytes() []byte
	Hash() valuehash.Hash // root hash
	PreviousBlock() valuehash.Hash
	Height() Height
	Round() Round
	Proposal() valuehash.Hash
	Operations() valuehash.Hash
	States() valuehash.Hash
	INITVoteproof() Voteproof
	ACCEPTVoteproof() Voteproof
	SetINITVoteproof(Voteproof) Block
	SetACCEPTVoteproof(Voteproof) Block
	CreatedAt() time.Time
}
