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
	lastBlock hash.Hash,
	nextHeight Height,
	nextBlock hash.Hash,
	nextRound Round,
	nextProposal hash.Hash,
) (Ballot, error) {
	ib := BaseBallotBody{
		node:      n,
		height:    nextHeight.Add(1),
		round:     nextRound + 1,
		proposal:  nextProposal,
		block:     nextBlock,
		lastBlock: lastBlock,
		stage:     StageINIT,
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
