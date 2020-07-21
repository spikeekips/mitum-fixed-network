// +build test

package base

import (
	"crypto/rand"
	"encoding/hex"
)

func MustStringAddress(s string) StringAddress {
	a, err := NewStringAddress(s)
	if err != nil {
		panic(err)
	}

	return a
}

func RandomStringAddress() StringAddress {
	b := make([]byte, 10)
	_, _ = rand.Read(b)

	return MustStringAddress(hex.EncodeToString(b))
}
