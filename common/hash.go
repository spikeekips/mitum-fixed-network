package common

import (
	"crypto/sha256"
	"encoding/json"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/rlp"
)

type NetworkID []byte

type Signature []byte

func NewSignature(networkID NetworkID, seed Seed, hash Hash) (Signature, error) {
	return seed.Sign(append(networkID, hash.Bytes()...))
}

func (s Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(base58.Encode(s[:]))
}

func (s *Signature) UnmarshalJSON(b []byte) error {
	var n string
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}

	*s = Signature(base58.Decode(n))

	return nil
}

type Hashable interface {
	Hash() (Hash, error)
}

func RawHash(b []byte) [32]byte {
	first := sha256.Sum256(b)
	return sha256.Sum256(first[:])
}

func RawHashFromObject(i interface{}) ([32]byte, error) {
	b, err := rlp.EncodeToBytes(i)
	if err != nil {
		return [32]byte{}, err
	}

	return RawHash(b), nil
}

type Hash struct {
	p string
	b [32]byte
}

func NewHash(prefix string, b []byte) Hash {
	return Hash{p: prefix, b: RawHash(b)}
}

func NewHashFromObject(prefix string, i interface{}) (Hash, error) {
	r, err := RawHashFromObject(i)
	if err != nil {
		return Hash{}, err
	}

	return Hash{p: prefix, b: r}, nil
}

func (h Hash) Prefix() string {
	return h.p
}

func (h Hash) Body() [32]byte {
	return h.b
}

func (h Hash) Bytes() []byte {
	return h.b[:]
}

func (h Hash) String() string {
	return h.Prefix() + "-" + base58.Encode(h.Bytes())
}

func (h Hash) Equal(n Hash) bool {
	return h.Prefix() == n.Prefix() && h.Body() == n.Body()
}

func (h Hash) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.String())
}

func (h *Hash) UnmarshalJSON(b []byte) error {
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
		return JSONUnmarshalError
	}

	var a [32]byte
	copy(a[:], decoded)

	h.p = s[0]
	h.b = a

	return nil
}
