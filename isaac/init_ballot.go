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
	previousBlock hash.Hash,
	newBlock hash.Hash,
	newRound Round,
	newProposal hash.Hash,
	nextHeight Height,
	nextRound Round,
) (Ballot, error) {
	ib := BaseBallotBody{
		node:      n,
		stage:     StageINIT,
		height:    nextHeight,    // next height
		round:     nextRound,     // next round
		proposal:  newProposal,   // proposal for new block
		block:     newBlock,      // block for new block
		lastBlock: previousBlock, // previous block
		lastRound: newRound,      // round for new block
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
