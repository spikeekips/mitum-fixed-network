package valuehash

import (
	"bytes"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/mitum/util/isvalid"
)

const maxBytesHashSize = 100

type Bytes []byte

func NewBytes(b []byte) Bytes {
	return Bytes(b)
}

func NewBytesFromString(s string) Bytes {
	return NewBytes(base58.Decode(s))
}

func (hs Bytes) String() string {
	return toString(hs)
}

func (hs Bytes) IsEmpty() bool {
	return hs == nil || len(hs) < 1
}

func (hs Bytes) IsValid([]byte) error {
	if hs.IsEmpty() {
		return isvalid.InvalidError.Wrap(EmptyHashError)
	} else if len(hs) > maxBytesHashSize {
		return isvalid.InvalidError.Wrap(InvalidHashError.Errorf(
			"over max: %d > %d", len(hs), maxBytesHashSize))
	}

	return nil
}

func (hs Bytes) Bytes() []byte {
	return []byte(hs)
}

func (hs Bytes) Equal(h Hash) bool {
	return bytes.Equal(hs, h.Bytes())
}
