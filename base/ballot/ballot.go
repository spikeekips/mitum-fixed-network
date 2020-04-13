package ballot

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
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
	Stage() base.Stage
	Height() base.Height
	Round() base.Round
	Node() base.Address
}

type INITBallot interface {
	Ballot
	PreviousBlock() valuehash.Hash
	PreviousRound() base.Round
	Voteproof() base.Voteproof
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
	Voteproof() base.Voteproof
}

type INITBallotFact interface {
	valuehash.Hasher
	PreviousBlock() valuehash.Hash
	PreviousRound() base.Round
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

func IsValidBallot(ballot Ballot, b []byte) error {
	if err := seal.IsValidSeal(ballot, b); err != nil {
		return err
	}

	return operation.IsValidEmbededFact(ballot.Signer(), ballot, b)
}
