package isaac

import (
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/node"
)

type ACCEPTBallotBody struct {
	BaseBallotBody
}

func NewACCEPTBallot(
	n node.Address,
	lastBlock hash.Hash,
	nextHeight Height,
	nextBlock hash.Hash,
	currentRound Round,
	currentProposal hash.Hash,
) (Ballot, error) {
	ib := BaseBallotBody{
		node:      n,
		height:    nextHeight,
		round:     currentRound,
		proposal:  currentProposal,
		block:     nextBlock,
		lastBlock: lastBlock,
		stage:     StageACCEPT,
	}

	h, err := ib.makeHash()
	if err != nil {
		return Ballot{}, err
	}

	ib.hash = h

	ballot, err := NewBallot(ACCEPTBallotBody{BaseBallotBody: ib})
	if err != nil {
		return Ballot{}, err
	}

	return ballot, nil
}
