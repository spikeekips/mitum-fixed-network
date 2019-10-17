package contest_module

import (
	"crypto/rand"
	mrand "math/rand"

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

func NewRandomBlockHash() hash.Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	h, _ := isaac.NewBlockHash(b)

	return h
}

func NewRandomBlock() isaac.Block {
	bk, _ := isaac.NewBlock(
		NewRandomHeight(),
		NewRandomRound(),
		NewRandomProposalHash(),
	)

	return bk
}

func NewRandomNextBlock(bk isaac.Block) isaac.Block {
	nbk, _ := isaac.NewBlock(
		bk.Height().Add(1),
		NewRandomRound(),
		NewRandomProposalHash(),
	)

	return nbk
}

func NewRandomHeight() isaac.Height {
	return isaac.NewBlockHeight(uint64(mrand.Intn(100)))
}

func NewRandomRound() isaac.Round {
	return isaac.Round(uint64(mrand.Intn(100)))
}
