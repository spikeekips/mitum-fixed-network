package valuehash

import (
	"encoding/hex"

	"golang.org/x/crypto/sha3"

	"github.com/spikeekips/mitum/hint"
)

const sha256Size int = 32
const sha512Size int = 64

var emptySHA256 [sha256Size]byte
var emptySHA512 [sha512Size]byte
var nilSHA256 [sha256Size]byte
var nilSHA512 [sha512Size]byte

var sha256Hint = hint.MustHint(hint.Type{0x07, 0x00}, "0.1")
var sha512Hint = hint.MustHint(hint.Type{0x07, 0x01}, "0.1")

func init() {
	nilSHA256 = sha3.Sum256(nil)
	nilSHA512 = sha3.Sum512(nil)
}

type SHA512 struct {
	b [sha512Size]byte
}

func NewSHA512(b []byte) Hash {
	return SHA512{b: sha3.Sum512(b)}
}

func (s512 SHA512) String() string {
	return hex.EncodeToString(s512.Bytes())
}

func (s512 SHA512) Hint() hint.Hint {
	return sha512Hint
}

func (s512 SHA512) IsValid([]byte) error {
	if emptySHA512 == s512.b || nilSHA512 == s512.b {
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

type SHA256 struct {
	b [sha256Size]byte
}

func NewSHA256(b []byte) Hash {
	return SHA256{b: sha3.Sum256(b)}
}

func (s256 SHA256) String() string {
	return hex.EncodeToString(s256.Bytes())
}

func (s256 SHA256) Hint() hint.Hint {
	return sha256Hint
}

func (s256 SHA256) IsValid([]byte) error {
	if emptySHA256 == s256.b || nilSHA256 == s256.b {
		return EmptyHashError
	}

	return nil
}

func (s256 SHA256) Size() int {
	return sha256Size
}

func (s256 SHA256) Bytes() []byte {
	return s256.b[:]
}

func (s256 SHA256) Equal(h Hash) bool {
	if s256.Hint().Type() != h.Hint().Type() {
		return false
	}
	if s256.b != h.(SHA256).b {
		return false
	}

	return true
}
