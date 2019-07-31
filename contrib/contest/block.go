package main

import (
	"crypto/rand"

	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
)

func NewRandomProposalHash() hash.Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	h, _ := isaac.NewProposalHash(b)
	return h
}

func NewBlock(height isaac.Height, round isaac.Round) isaac.Block {
	bk, _ := isaac.NewBlock(
		height,
		round,
		NewRandomProposalHash(),
	)

	return bk
}
