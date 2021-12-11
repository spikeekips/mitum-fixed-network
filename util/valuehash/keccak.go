package valuehash

import (
	"bytes"

	"github.com/spikeekips/mitum/util/isvalid"
	"golang.org/x/crypto/sha3"
)

const (
	sha256Size int = 32
	sha512Size int = 64
)

var (
	emptySHA256 [sha256Size]byte
	nilSHA256   [sha256Size]byte
	emptySHA512 [sha512Size]byte
	nilSHA512   [sha512Size]byte
)

func init() {
	nilSHA256 = sha3.Sum256(nil)
	nilSHA512 = sha3.Sum512(nil)
}

type SHA512 struct {
	b [sha512Size]byte
}

func NewSHA512(b []byte) SHA512 {
	return SHA512{b: sha3.Sum512(b)}
}

func (hs SHA512) String() string {
	return toString(hs.b[:])
}

func (hs SHA512) IsEmpty() bool {
	return emptySHA512 == hs.b || nilSHA512 == hs.b
}

func (hs SHA512) IsValid([]byte) error {
	if hs.IsEmpty() {
		return isvalid.InvalidError.Wrap(EmptyHashError)
	}

	return nil
}

func (hs SHA512) Bytes() []byte {
	return hs.b[:]
}

func (hs SHA512) Equal(h Hash) bool {
	return bytes.Equal(hs.b[:], h.Bytes())
}

type SHA256 struct {
	b [sha256Size]byte
}

func NewSHA256(b []byte) SHA256 {
	return SHA256{b: sha3.Sum256(b)}
}

func (hs SHA256) String() string {
	return toString(hs.b[:])
}

func (hs SHA256) IsEmpty() bool {
	return emptySHA256 == hs.b || nilSHA256 == hs.b
}

func (hs SHA256) IsValid([]byte) error {
	if hs.IsEmpty() {
		return isvalid.InvalidError.Wrap(EmptyHashError)
	}

	return nil
}

func (hs SHA256) Bytes() []byte {
	return hs.b[:]
}

func (hs SHA256) Equal(h Hash) bool {
	return bytes.Equal(hs.b[:], h.Bytes())
}
