package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	INITBallotType         hint.Type = hint.Type{0x03, 0x00}
	ProposalBallotType     hint.Type = hint.Type{0x03, 0x01}
	SIGNBallotType         hint.Type = hint.Type{0x03, 0x02}
	ACCEPTBallotType       hint.Type = hint.Type{0x03, 0x03}
	INITBallotFactType     hint.Type = hint.Type{0x03, 0x04}
	ProposalBallotFactType hint.Type = hint.Type{0x03, 0x05}
	SIGNBallotFactType     hint.Type = hint.Type{0x03, 0x06}
	ACCEPTBallotFactType   hint.Type = hint.Type{0x03, 0x07}
)

type Ballot interface {
	FactSeal
	Stage() Stage
	Height() Height
	Round() Round
	Node() Address
}

type INITBallot interface {
	Ballot
	PreviousBlock() valuehash.Hash
	PreviousRound() Round
	Voteproof() Voteproof
}

type Proposal interface {
	Ballot
	Seals() []valuehash.Hash // collection of received Seals
}

type SIGNBallot interface {
	Ballot
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
}

type ACCEPTBallot interface {
	Ballot
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
	Voteproof() Voteproof
}

type INITBallotFact interface {
	Hash([]byte) (valuehash.Hash, error)
	PreviousBlock() valuehash.Hash
	PreviousRound() Round
}

type SIGNBallotFact interface {
	Hash([]byte) (valuehash.Hash, error)
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
}

type ACCEPTBallotFact interface {
	Hash([]byte) (valuehash.Hash, error)
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
}
