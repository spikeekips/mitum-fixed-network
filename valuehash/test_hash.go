// +build test

package valuehash

import "crypto/rand"

func RandomSHA512() Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	return NewSHA512(b)
}

func RandomSHA256() Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	return NewSHA256(b)
}
