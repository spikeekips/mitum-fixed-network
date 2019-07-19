// +build test

package seal

import (
	"crypto/rand"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
)

func init() {
	common.SetTestLogger(Log())
}

func NewRandomSealHash() hash.Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	h, _ := hash.NewHash(SealHashHint, b)
	return h
}
