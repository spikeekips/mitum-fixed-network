package base

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	INITBallotType       = hint.Type("init-ballot")
	ProposalType         = hint.Type("proposal")
	ACCEPTBallotType     = hint.Type("accept-ballot")
	INITBallotFactType   = hint.Type("init-ballot-fact")
	ProposalFactType     = hint.Type("proposal-fact")
	ACCEPTBallotFactType = hint.Type("accept-ballot-fact")
)

type Ballot interface {
	zerolog.LogObjectMarshaler
	seal.Seal
	RawFact() BallotFact
	FactSign() BallotFactSign
	SignedFact() SignedBallotFact
	BaseVoteproof() Voteproof
	ACCEPTVoteproof() Voteproof
}

type BallotFact interface {
	Fact
	Stage() Stage
	Height() Height
	Round() Round
}

type BallotFactSign interface {
	FactSign
	Node() Address
}

type INITBallotFact interface {
	BallotFact
	PreviousBlock() valuehash.Hash
}

type ProposalFact interface {
	BallotFact
	Proposer() Address
	Operations() []valuehash.Hash
	ProposedAt() time.Time
}

type ACCEPTBallotFact interface {
	BallotFact
	Proposal() valuehash.Hash // NOTE fact hash of proposal ballot
	NewBlock() valuehash.Hash
}

type INITBallot interface {
	Ballot
	Fact() INITBallotFact
}

type Proposal interface {
	Ballot
	Fact() ProposalFact
}

type ACCEPTBallot interface {
	Ballot
	Fact() ACCEPTBallotFact
}

type SignWithFacter interface {
	Sign(key.Privatekey, []byte) error
	SignWithFact(Address, key.Privatekey, []byte) error
}
