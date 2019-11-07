package isaac

import (
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
)

type BallotMaker interface {
	INIT(
		lastBlock hash.Hash,
		lastRound Round,
		nextHeight Height,
		nextBlock hash.Hash,
		currentRound Round,
		currentProposal hash.Hash,
	) (Ballot, error) // NOTE signed seal
	SIGN(
		lastBlock hash.Hash,
		lastRound Round,
		nextHeight Height,
		nextBlock hash.Hash,
		currentRound Round,
		currentProposal hash.Hash,
	) (Ballot, error) // NOTE signed seal
	ACCEPT(
		lastBlock hash.Hash,
		lastRound Round,
		nextHeight Height,
		nextBlock hash.Hash,
		currentRound Round,
		currentProposal hash.Hash,
	) (Ballot, error) // NOTE signed seal
}

type DefaultBallotMaker struct {
	*common.Logger
	home node.Home
}

func NewDefaultBallotMaker(home node.Home) DefaultBallotMaker {
	return DefaultBallotMaker{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "default-ballot_maker")
		}),
		home: home,
	}
}

func (db DefaultBallotMaker) INIT(
	lastBlock hash.Hash,
	lastRound Round,
	nextHeight Height,
	nextBlock hash.Hash,
	currentRound Round,
	currentProposal hash.Hash,
) (Ballot, error) {
	ballot, err := NewINITBallot(
		db.home.Address(), lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
	)
	if err != nil {
		return Ballot{}, err
	}

	return db.sign(ballot)
}

func (db DefaultBallotMaker) SIGN(
	lastBlock hash.Hash,
	lastRound Round,
	nextHeight Height,
	nextBlock hash.Hash,
	currentRound Round,
	currentProposal hash.Hash,
) (Ballot, error) {
	ballot, err := NewSIGNBallot(
		db.home.Address(), lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
	)
	if err != nil {
		return Ballot{}, err
	}

	return db.sign(ballot)
}

func (db DefaultBallotMaker) ACCEPT(
	lastBlock hash.Hash,
	lastRound Round,
	nextHeight Height,
	nextBlock hash.Hash,
	currentRound Round,
	currentProposal hash.Hash,
) (Ballot, error) {
	ballot, err := NewACCEPTBallot(
		db.home.Address(), lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
	)
	if err != nil {
		return Ballot{}, err
	}

	return db.sign(ballot)
}

func (db DefaultBallotMaker) sign(ballot Ballot) (Ballot, error) {
	if err := ballot.Sign(db.home.PrivateKey(), nil); err != nil {
		return Ballot{}, err
	}

	return ballot, nil
}
