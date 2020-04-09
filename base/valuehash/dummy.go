package valuehash

import (
	"bytes"
	"encoding/hex"

	"github.com/spikeekips/mitum/util/hint"
)

var (
	dummyType = hint.MustNewType(0x07, 0x02, "hash-dummy")
	dummyHint = hint.MustHint(dummyType, "0.0.1")
)

type Dummy struct {
	b []byte
}

func NewDummy(b []byte) Dummy {
	return Dummy{b: b}
}

func (dm Dummy) String() string {
	return hex.EncodeToString(dm.b)
}

func (dm Dummy) Hint() hint.Hint {
	return dummyHint
}

func (dm Dummy) IsValid([]byte) error {
	if len(dm.b) < 1 {
		return EmptyHashError
	}

	return nil
}

func (dm Dummy) Size() int {
	return len(dm.b)
}

func (dm Dummy) Bytes() []byte {
	return dm.b
}

func (dm Dummy) Equal(h Hash) bool {
	if dm.Hint().Type() != h.Hint().Type() {
		return false
	}
	return bytes.Equal(dm.b, h.Bytes())
}
