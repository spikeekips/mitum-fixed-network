package valuehash

import (
	"bytes"

	"github.com/zeebo/blake3"
)

const (
	blake3256Size int = 32
)

var (
	emptyBlake3256 [blake3256Size]byte
	nilBlake3256   [blake3256Size]byte
)

func init() {
	nilBlake3256 = blake3.Sum256(nil)
}

type Blake3256 struct {
	b [blake3256Size]byte
}

func NewBlake3256(b []byte) Blake3256 {
	return Blake3256{b: blake3.Sum256(b)}
}

func (hs Blake3256) String() string {
	return toString(hs.b[:])
}

func (hs Blake3256) IsEmpty() bool {
	return emptyBlake3256 == hs.b || nilBlake3256 == hs.b
}

func (hs Blake3256) IsValid([]byte) error {
	if hs.IsEmpty() {
		return EmptyHashError.Call()
	}

	return nil
}

func (hs Blake3256) Bytes() []byte {
	return hs.b[:]
}

func (hs Blake3256) Equal(h Hash) bool {
	return bytes.Equal(hs.b[:], h.Bytes())
}
