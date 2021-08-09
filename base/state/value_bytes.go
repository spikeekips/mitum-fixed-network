package state

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	BytesValueType = hint.Type("state-bytes-value")
	BytesValueHint = hint.NewHint(BytesValueType, "v0.0.1")
)

type BytesValue struct {
	v []byte
	h valuehash.Hash
}

func NewBytesValue(v interface{}) (BytesValue, error) {
	return BytesValue{}.set(v)
}

func (BytesValue) set(v interface{}) (BytesValue, error) {
	var s []byte
	switch t := v.(type) {
	case []byte:
		s = t
	case string:
		s = []byte(t)
	case util.Byter:
		if v != nil {
			s = t.Bytes()
		}
	case fmt.Stringer:
		if v != nil {
			s = []byte(t.String())
		}
	default:
		return BytesValue{}, errors.Errorf("not bytes-like: %T", v)
	}

	return BytesValue{
		v: s,
		h: valuehash.NewSHA256(s),
	}, nil
}

func (bv BytesValue) IsValid([]byte) error {
	return bv.h.IsValid(nil)
}

func (bv BytesValue) Bytes() []byte {
	return bv.v
}

func (BytesValue) Hint() hint.Hint {
	return BytesValueHint
}

func (bv BytesValue) Equal(v Value) bool {
	return bv.Hash().Equal(v.Hash())
}

func (bv BytesValue) Hash() valuehash.Hash {
	return bv.h
}

func (bv BytesValue) Interface() interface{} {
	return bv.v
}

func (bv BytesValue) Set(v interface{}) (Value, error) {
	return bv.set(v)
}
