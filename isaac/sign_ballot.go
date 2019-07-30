package isaac

import (
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
)

type SIGNBallotBody struct {
	BaseBallotBody
}

func NewSIGNBallot(
	n node.Address,
	lastBlock hash.Hash,
	lastRound Round,
	nextHeight Height,
	nextBlock hash.Hash,
	currentRound Round,
	currentProposal hash.Hash,
) (Ballot, error) {
	ib := BaseBallotBody{
		node:      n,
		stage:     StageSIGN,
		height:    nextHeight,
		round:     currentRound,
		proposal:  currentProposal,
		block:     nextBlock,
		lastBlock: lastBlock,
		lastRound: lastRound,
	}

	h, err := ib.makeHash()
	if err != nil {
		return Ballot{}, err
	}

	ib.hash = h

	ballot, err := NewBallot(SIGNBallotBody{BaseBallotBody: ib})
	if err != nil {
		return Ballot{}, err
	}

	return ballot, nil
}
