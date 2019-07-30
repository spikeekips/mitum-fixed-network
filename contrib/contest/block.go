package main

import (
	"github.com/spikeekips/mitum/isaac"
)

func NewBlock(height isaac.Height, round isaac.Round) isaac.Block {
	bk, _ := isaac.NewBlock(
		height,
		round,
		isaac.NewRandomProposalHash(),
	)

	return bk
}
