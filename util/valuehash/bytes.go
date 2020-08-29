package valuehash

import (
	"bytes"

	"github.com/spikeekips/mitum/util/hint"
)

const maxBytesHashSize = 100

var (
	bytesType = hint.MustNewType(0x01, 0x80, "hash-bytes")
	bytesHint = hint.MustHint(bytesType, "0.0.1")
)

type Bytes []byte

func NewBytes(b []byte) Bytes {
	return Bytes(b)
}

func NewBytesFromString(s string) Bytes {
	return NewBytes(fromString(s))
}

func (hs Bytes) String() string {
	return toString(hs)
}

func (hs Bytes) Hint() hint.Hint {
	return bytesHint
}

func (hs Bytes) Empty() bool {
	return hs == nil || len(hs) < 1
}

func (hs Bytes) IsValid([]byte) error {
	if hs.Empty() {
		return EmptyHashError
	} else if len(hs) > maxBytesHashSize {
		return InvalidHashError.Errorf("over max: %d > %d", len(hs), maxBytesHashSize)
	}

	return nil
}

func (hs Bytes) Size() int {
	return len(hs)
}

func (hs Bytes) Bytes() []byte {
	return []byte(hs)
}

func (hs Bytes) Equal(h Hash) bool {
	return bytes.Equal(hs, h.Bytes())
}
