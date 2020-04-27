package state

import (
	"fmt"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	StringValueType = hint.MustNewType(0x12, 0x01, "state-string-value")
	StringValueHint = hint.MustHint(StringValueType, "0.0.1")
)

type StringValue struct {
	v string
	h valuehash.Hash
}

func NewStringValue(v interface{}) (StringValue, error) {
	return StringValue{}.set(v)
}

func (sv StringValue) set(v interface{}) (StringValue, error) {
	var s string
	switch t := v.(type) {
	case string:
		s = t
	case fmt.Stringer:
		if v != nil {
			s = t.String()
		}
	default:
		return StringValue{}, xerrors.Errorf("not string-like: %T", v)
	}

	return StringValue{
		v: s,
		h: valuehash.NewSHA256([]byte(s)),
	}, nil
}

func (sv StringValue) IsValid([]byte) error {
	if err := sv.h.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (sv StringValue) Bytes() []byte {
	return []byte(sv.v)
}

func (sv StringValue) Hint() hint.Hint {
	return StringValueHint
}

func (sv StringValue) Equal(v Value) bool {
	return sv.Hash().Equal(v.Hash())
}

func (sv StringValue) Hash() valuehash.Hash {
	return sv.h
}

func (sv StringValue) Interface() interface{} {
	return sv.v
}

func (sv StringValue) Set(v interface{}) (Value, error) {
	return sv.set(v)
}
