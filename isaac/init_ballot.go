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
	lastRound Round,
	nextHeight Height,
	nextBlock hash.Hash,
	currentRound Round,
	currentProposal hash.Hash,
) (Ballot, error) {
	ib := BaseBallotBody{
		node:      n,
		stage:     StageINIT,
		height:    nextHeight,      // next height
		round:     currentRound,    // next round
		proposal:  currentProposal, // proposal for new block
		block:     nextBlock,       // block for new block
		lastBlock: lastBlock,       // previous block
		lastRound: lastRound,       // round for new block
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
