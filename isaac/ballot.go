package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	INITBallotType         = hint.MustNewType(0x03, 0x00, "init-ballot")
	ProposalBallotType     = hint.MustNewType(0x03, 0x01, "proposal")
	SIGNBallotType         = hint.MustNewType(0x03, 0x02, "sign-ballot")
	ACCEPTBallotType       = hint.MustNewType(0x03, 0x03, "accept-ballot")
	INITBallotFactType     = hint.MustNewType(0x03, 0x04, "init-ballot-fact")
	ProposalBallotFactType = hint.MustNewType(0x03, 0x05, "proposal-fact")
	SIGNBallotFactType     = hint.MustNewType(0x03, 0x06, "sign-ballot-fact")
	ACCEPTBallotFactType   = hint.MustNewType(0x03, 0x07, "accept-ballot-fact")
)

type Ballot interface {
	operation.FactSeal
	logging.LogHintedMarshaler
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
	Operations() []valuehash.Hash // collection of proposed Operations
	Seals() []valuehash.Hash      // NOTE collection of received Seals, which must have Operations()
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
	valuehash.Hasher
	PreviousBlock() valuehash.Hash
	PreviousRound() Round
}

type SIGNBallotFact interface {
	valuehash.Hasher
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
}

type ACCEPTBallotFact interface {
	valuehash.Hasher
	Proposal() valuehash.Hash
	NewBlock() valuehash.Hash
}
