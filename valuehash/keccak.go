package valuehash

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"

	"github.com/spikeekips/mitum/hint"
)

const sha512Size int = 64

var emptySHA512 [64]byte
var sha512Hint = hint.MustHint(hint.Type{0x02, 0x00}, "0.1")

func init() {
	copy(emptySHA512[:], sha3.New512().Sum(nil))
}

type SHA512 struct {
	b [sha512Size]byte
}

func NewSHA512(b []byte) Hash {
	var s [sha512Size]byte
	copy(s[:], sha3.New512().Sum(b))

	return SHA512{b: s}
}

func (s512 SHA512) String() string {
	return hex.EncodeToString(s512.Bytes())
}

func (s512 SHA512) Hint() hint.Hint {
	return sha512Hint
}

func (s512 SHA512) IsValid() error {
	if emptySHA512 == s512.b {
		return EmptyHashError
	}

	return nil
}

func (s512 SHA512) Size() int {
	return sha512Size
}

func (s512 SHA512) Bytes() []byte {
	return s512.b[:]
}

func (s512 SHA512) Equal(h Hash) bool {
	if s512.Hint().Type() != h.Hint().Type() {
		return false
	}
	if s512.b != h.(SHA512).b {
		return false
	}

	return true
}
