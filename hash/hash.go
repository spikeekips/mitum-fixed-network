package hash

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/rs/zerolog"
)

var (
	zeroBody [100]byte = [100]byte{}
	nilBody  [5]byte   = [5]byte{186, 47, 126, 25, 238}
)

type Hash struct {
	hint   string
	body   [100]byte // NOTE the fixed length array can be possible to make Hash to be comparable
	length int
}

func NewHash(hint string, body []byte) (Hash, error) {
	if len(hint) < 1 {
		return Hash{}, HashFailedError.Newf("zero hint length")
	}

	var b [100]byte
	copy(b[:], body)

	return Hash{
		hint:   hint,
		body:   b,
		length: len(body),
	}, nil
}

func NilHash(hint string) Hash {
	h, _ := NewHash(hint, nilBody[:]) // nolint
	return h
}

func (h Hash) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, struct {
		Hint string
		Body []byte
	}{
		h.hint,
		h.Body(),
	})
}

func (h *Hash) DecodeRLP(s *rlp.Stream) error {
	var d struct {
		Hint string
		Body []byte
	}
	if err := s.Decode(&d); err != nil {
		return InvalidHashInputError.New(err)
	}

	h.hint = d.Hint
	var b [100]byte
	copy(b[:], d.Body)

	h.body = b
	h.length = len(d.Body)

	return nil
}

func (h Hash) Empty() bool {
	if len(h.hint) > 0 || h.body != zeroBody {
		return false
	}

	return true
}

func (h Hash) IsNil() bool {
	if len(h.hint) < 1 || h.length != len(nilBody) {
		return false
	}

	for i, a := range nilBody {
		if a != h.body[i] {
			return false
		}
	}

	return true
}

func (h Hash) IsValid() error {
	if h.length < 1 {
		return EmptyHashError.Newf("empty body")
	}

	if len(h.hint) < 1 {
		return EmptyHashError.Newf("empty hint")
	}

	return nil
}

func (h Hash) Equal(a Hash) bool {
	if h.hint != a.hint {
		return false
	}
	if h.body != a.body {
		return false
	}

	for i, b := range h.Body() {
		if b != a.body[i] {
			return false
		}
	}

	return true
}

func (h Hash) MarshalJSON() ([]byte, error) {
	/* NOTE
	return json.Marshal(map[string]interface{}{
		"hint": h.hint,
		"body": base58.Encode(h.Body()),
	})
	*/
	return json.Marshal(h.String())
}

func (h Hash) MarshalZerologObject(e *zerolog.Event) {
	e.Str("hash", h.String())
}

func (h Hash) MarshalText() ([]byte, error) {
	return []byte(h.String()), nil
}

func (h Hash) Hint() string {
	return h.hint
}

func (h Hash) Body() []byte {
	return h.body[:h.length]
}

func (h Hash) Bytes() []byte {
	var n []byte
	n = append(n, []byte(h.hint)...)
	n = append(n, h.body[:h.length]...)

	return n
}

func (h Hash) String() string {
	return fmt.Sprintf("%s:%s", h.hint, base58.Encode(h.Body()))
}
