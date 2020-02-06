// +build test

package isaac

import "github.com/spikeekips/mitum/valuehash"

func NewTestBlockV0(height Height, round Round, proposal, previousBlock valuehash.Hash) (BlockV0, error) {
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
		nil,
		nil,
		nil,
	)
}
