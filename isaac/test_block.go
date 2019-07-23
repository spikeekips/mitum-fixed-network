// +build test

package isaac

import (
	"crypto/rand"
	"math/big"

	"github.com/spikeekips/mitum/hash"
)

func NewRandomProposalHash() hash.Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	h, _ := NewProposalHash(b)
	return h
}

func NewRandomBlock() Block {
	b, _ := rand.Int(rand.Reader, big.NewInt(27))

	bk, _ := NewBlock(
		NewBlockHeight(uint64(b.Int64())),
		Round(uint64(b.Int64())),
		NewRandomProposalHash(),
	)

	return bk
}

func NewRandomNextBlock(bk Block) Block {
	nbk, _ := NewBlock(
		bk.Height().Add(1),
		bk.Round()+1,
		NewRandomProposalHash(),
	)

	return nbk
}

func NewRandomBlockHash() hash.Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	h, _ := NewBlockHash(b)
	return h
}
