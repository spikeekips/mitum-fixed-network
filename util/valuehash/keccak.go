package valuehash

import (
	"bytes"

	"github.com/pkg/errors"
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

func LoadSHA512FromBytes(b []byte) (Hash, error) {
	if l := len(b); l != sha512Size {
		return nil, errors.Errorf("invalid sha512 size: %d", l)
	}

	n := [sha512Size]byte{}
	copy(n[:], b)

	return SHA512{b: n}, nil
}

func LoadSHA512FromString(s string) (Hash, error) {
	return LoadSHA512FromBytes(fromString(s))
}

func (hs SHA512) String() string {
	return toString(hs.b[:])
}

func (hs SHA512) Empty() bool {
	return emptySHA512 == hs.b || nilSHA512 == hs.b
}

func (hs SHA512) IsValid([]byte) error {
	if emptySHA512 == hs.b || nilSHA512 == hs.b {
		return EmptyHashError
	}

	return nil
}

func (SHA512) Size() int {
	return sha512Size
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

func LoadSHA256FromBytes(b []byte) (Hash, error) {
	if l := len(b); l != sha256Size {
		return nil, errors.Errorf("invalid sha256 size: %d", l)
	}

	n := [sha256Size]byte{}
	copy(n[:], b)

	return SHA256{b: n}, nil
}

func LoadSHA256FromString(s string) (Hash, error) {
	return LoadSHA256FromBytes(fromString(s))
}

func (hs SHA256) String() string {
	return toString(hs.b[:])
}

func (hs SHA256) Empty() bool {
	return emptySHA256 == hs.b || nilSHA256 == hs.b
}

func (hs SHA256) IsValid([]byte) error {
	if emptySHA256 == hs.b || nilSHA256 == hs.b {
		return EmptyHashError
	}

	return nil
}

func (SHA256) Size() int {
	return sha256Size
}

func (hs SHA256) Bytes() []byte {
	return hs.b[:]
}

func (hs SHA256) Equal(h Hash) bool {
	return bytes.Equal(hs.b[:], h.Bytes())
}
