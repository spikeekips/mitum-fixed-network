package common

import (
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/valuehash"
)

func NewContestBlock(
	height isaac.Height,
	round isaac.Round,
	proposal,
	previousBlock valuehash.Hash,
) (isaac.BlockV0, error) {
	if proposal == nil {
		proposal = valuehash.RandomSHA256()
	}

	return isaac.NewBlockV0(
		height, round, proposal, previousBlock,
		nil, nil, nil,
	)
}
