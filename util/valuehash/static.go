package valuehash

import (
	"bytes"

	"github.com/spikeekips/mitum/util/isvalid"
)

type (
	L32 [32]byte
	L64 [64]byte
)

var (
	emptyL32 [32]byte
	emptyL64 [64]byte
)

func (h L32) IsValid([]byte) error {
	if h.IsEmpty() {
		return isvalid.InvalidError.Wrap(EmptyHashError)
	}

	return nil
}

func (h L32) Bytes() []byte {
	return h[:]
}

func (h L32) String() string {
	return toString(h[:])
}

func (h L32) Equal(b Hash) bool {
	return bytes.Equal(h[:], b.Bytes())
}

func (h L32) IsEmpty() bool {
	return emptyL32 == h
}

func (h L64) IsValid([]byte) error {
	if h.IsEmpty() {
		return isvalid.InvalidError.Wrap(EmptyHashError)
	}

	return nil
}

func (h L64) Bytes() []byte {
	return h[:]
}

func (h L64) String() string {
	return toString(h[:])
}

func (h L64) Equal(b Hash) bool {
	return bytes.Equal(h[:], b.Bytes())
}

func (h L64) IsEmpty() bool {
	return emptyL64 == h
}
