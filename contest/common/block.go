package common

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/isaac"
)

func NewContestBlock(
	height base.Height,
	round base.Round,
	proposal,
	previousBlock valuehash.Hash,
) (isaac.BlockV0, error) {
	if proposal == nil {
		proposal = valuehash.RandomSHA256()
	}

	return isaac.NewBlockV0(height, round, proposal, previousBlock, nil, nil)
}
