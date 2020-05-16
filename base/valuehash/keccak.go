package valuehash

import (
	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/crypto/sha3"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/hint"
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

var (
	sha256Type = hint.MustNewType(0x07, 0x00, "hash-sha256")
	sha256Hint = hint.MustHint(sha256Type, "0.0.1")
	sha512Type = hint.MustNewType(0x07, 0x01, "hash-sha512")
	sha512Hint = hint.MustHint(sha512Type, "0.0.1")
)

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

func LoadSHA512FromBytes(b []byte) (Hash, error) {
	if l := len(b); l != sha512Size {
		return nil, xerrors.Errorf("invalid sha512 size: %d", l)
	}

	n := [sha512Size]byte{}
	copy(n[:], b)

	return SHA512{b: n}, nil
}

func LoadSHA512FromString(s string) (Hash, error) {
	return LoadSHA512FromBytes(base58.Decode(s))
}

func (s512 SHA512) String() string {
	return base58.Encode(s512.Bytes())
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

func LoadSHA256FromBytes(b []byte) (Hash, error) {
	if l := len(b); l != sha256Size {
		return nil, xerrors.Errorf("invalid sha256 size: %d", l)
	}

	n := [sha256Size]byte{}
	copy(n[:], b)

	return SHA256{b: n}, nil
}

func LoadSHA256FromString(s string) (Hash, error) {
	return LoadSHA256FromBytes(base58.Decode(s))
}

func (s256 SHA256) String() string {
	return base58.Encode(s256.Bytes())
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
