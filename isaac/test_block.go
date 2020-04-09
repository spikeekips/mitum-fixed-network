// +build test

package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/valuehash"
)

func NewTestBlockV0(height base.Height, round base.Round, proposal, previousBlock valuehash.Hash) (BlockV0, error) {
	if proposal == nil {
		proposal = valuehash.RandomSHA256()
	}

	return NewBlockV0(
		height,
		round,
		proposal,
		previousBlock,
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	)
}
