package state

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	StringValueType   = hint.Type("state-string-value")
	StringValueHint   = hint.NewHint(StringValueType, "v0.0.1")
	StringValueHinter = StringValue{BaseHinter: hint.NewBaseHinter(StringValueHint)}
)

type StringValue struct {
	hint.BaseHinter
	v string
	h valuehash.Hash
}

func NewStringValue(v interface{}) (StringValue, error) {
	return StringValue{BaseHinter: hint.NewBaseHinter(StringValueHint)}.set(v)
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
		return StringValue{}, errors.Errorf("not string-like: %T", v)
	}

	sv.v = s
	sv.h = valuehash.NewSHA256([]byte(s))

	return sv, nil
}

func (sv StringValue) IsValid([]byte) error {
	return isvalid.Check(nil, false, sv.BaseHinter, sv.h)
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
