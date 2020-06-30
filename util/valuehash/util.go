package valuehash

import (
	"crypto/rand"

	"github.com/btcsuite/btcutil/base58"
)

func RandomSHA256() Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	return NewSHA256(b)
}

func RandomSHA512() Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	return NewSHA512(b)
}

func toString(b []byte) string {
	return base58.Encode(b)
}

func fromString(s string) []byte {
	return base58.Decode(s)
}
