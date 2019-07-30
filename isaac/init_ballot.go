package isaac

import (
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
)

type INITBallotBody struct {
	BaseBallotBody
}

func NewINITBallot(
	n node.Address,
	newBlock hash.Hash,
	newRound Round,
	nextHeight Height,
	nextBlock hash.Hash,
	nextRound Round,
	newProposal hash.Hash,
) (Ballot, error) {
	ib := BaseBallotBody{
		node:      n,
		stage:     StageINIT,
		height:    nextHeight.Add(1),
		round:     nextRound,
		proposal:  newProposal,
		block:     nextBlock,
		lastBlock: newBlock,
		lastRound: newRound,
	}

	h, err := ib.makeHash()
	if err != nil {
		return Ballot{}, err
	}

	ib.hash = h

	ballot, err := NewBallot(INITBallotBody{BaseBallotBody: ib})
	if err != nil {
		return Ballot{}, err
	}

	return ballot, nil
}
