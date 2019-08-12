// +build test

package isaac

import (
	"crypto/rand"

	"github.com/spikeekips/mitum/hash"
)

func NewRandomBallotHash() hash.Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	h, _ := NewBallotHash(b)
	return h
}
