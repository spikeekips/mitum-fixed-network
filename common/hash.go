package common

import (
	"crypto/sha256"
	"encoding"
	"encoding/json"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	emptyRawHash = [32]byte{}
)

type Hashable interface {
	encoding.BinaryMarshaler
	Hash() (Hash, []byte, error) // Hash().Body() == RawHash(Hashable.Encode())
}

func Encode(i interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(i)
}

func Decode(b []byte, i interface{}) error {
	return rlp.DecodeBytes(b, i)
}

func RawHash(b []byte) [32]byte {
	first := sha256.Sum256(b)
	return sha256.Sum256(first[:])
}

func RawHashFromObject(i interface{}) ([32]byte, error) {
	var b []byte
	switch i.(type) {
	case []byte:
		b = i.([]byte)
	default:
		if e, err := Encode(i); err != nil {
			return [32]byte{}, err
		} else {
			b = e
		}
	}

	return RawHash(b), nil
}

type Hash struct {
	h string
	b [32]byte
}

func NewHash(hint string, b []byte) (Hash, error) {
	if len([]byte(hint)) != 2 {
		return Hash{}, InvalidHashHintError
	}

	return Hash{h: hint, b: RawHash(b)}, nil
}

func NewHashFromObject(hint string, i interface{}) (Hash, error) {
	r, err := Encode(i)
	if err != nil {
		return Hash{}, err
	}

	return NewHash(hint, r)
}

func (h Hash) Hint() string {
	return h.h
}

func (h Hash) Body() [32]byte {
	return h.b
}

func (h Hash) Bytes() []byte {
	return h.b[:]
}

func (h Hash) String() string {
	return h.Hint() + "-" + base58.Encode(h.Bytes())
}

func (h Hash) Equal(n Hash) bool {
	return h.Hint() == n.Hint() && h.Body() == n.Body()
}

func (h Hash) MarshalBinary() ([]byte, error) {
	if h.b == emptyRawHash {
		return nil, EmptyHashError
	}

	return append([]byte(h.Hint()), h.Bytes()...), nil
}

func (h *Hash) UnmarshalBinary(b []byte) error {
	if len(b) != 34 {
		return InvalidHashError
	}

	h.h = string(b[:2])

	copy(h.b[:], b[2:])

	return nil
}

func (h Hash) MarshalText() ([]byte, error) {
	if h.b == emptyRawHash {
		return nil, EmptyHashError
	}

	return json.Marshal(h.String())
}

func (h *Hash) UnmarshalText(b []byte) error {
	var n string
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}

	s := strings.SplitN(n, "-", 2)
	if len(s) != 2 || len(s[0]) < 1 || len(s[1]) < 1 {
		return InvalidHashError
	}

	decoded := base58.Decode(s[1])
	if len(decoded) != 32 {
		return InvalidHashError
	}

	var a [32]byte
	copy(a[:], decoded)

	h.h = s[0]
	h.b = a

	return nil
}

type NetworkID []byte

type Signature []byte

func NewSignature(networkID NetworkID, seed Seed, hash Hash) (Signature, error) {
	return seed.Sign(append(networkID, hash.Bytes()...))
}

func (s Signature) MarshalText() ([]byte, error) {
	return json.Marshal(base58.Encode(s[:]))
}

func (s *Signature) UnmarshalText(b []byte) error {
	var n string
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}

	*s = Signature(base58.Decode(n))

	return nil
}
