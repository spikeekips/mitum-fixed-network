package isaac

import (
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
)

type BallotMaker interface {
	INIT(
		previousBlock hash.Hash,
		newBlock hash.Hash,
		newRound Round,
		newProposal hash.Hash,
		nextHeight Height,
		nextRound Round,
	) (Ballot, error) // NOTE signed Ballot
	SIGN(
		lastBlock hash.Hash,
		lastRound Round,
		nextHeight Height,
		nextBlock hash.Hash,
		currentRound Round,
		currentProposal hash.Hash,
	) (Ballot, error) // NOTE signed Ballot
	ACCEPT(
		lastBlock hash.Hash,
		lastRound Round,
		nextHeight Height,
		nextBlock hash.Hash,
		currentRound Round,
		currentProposal hash.Hash,
	) (Ballot, error) // NOTE signed Ballot
}

type DefaultBallotMaker struct {
	home node.Home
}

func NewDefaultBallotMaker(home node.Home) DefaultBallotMaker {
	return DefaultBallotMaker{home: home}
}

func (db DefaultBallotMaker) INIT(
	previousBlock hash.Hash,
	newBlock hash.Hash,
	newRound Round,
	newProposal hash.Hash,
	nextHeight Height,
	nextRound Round,
) (Ballot, error) {
	ballot, err := NewINITBallot(
		db.home.Address(), previousBlock, newBlock, newRound, newProposal, nextHeight, nextRound,
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
