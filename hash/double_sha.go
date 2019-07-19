package hash

import (
	"crypto/sha256"
)

func NewDoubleSHAHash(hint string, b []byte) (Hash, error) {
	f := sha256.Sum256(b)
	s := sha256.Sum256(f[:])

	return NewHash(hint, s[:])
}
