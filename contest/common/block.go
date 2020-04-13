package common

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/valuehash"
)

func NewContestBlock(
	height base.Height,
	round base.Round,
	proposal,
	previousBlock valuehash.Hash,
) (block.BlockV0, error) {
	if proposal == nil {
		proposal = valuehash.RandomSHA256()
	}

	return block.NewBlockV0(height, round, proposal, previousBlock, nil, nil)
}
