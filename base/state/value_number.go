package state

import (
	"reflect"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	NumberValueType   = hint.Type("state-number-value")
	NumberValueHint   = hint.NewHint(NumberValueType, "v0.0.1")
	NumberValueHinter = NumberValue{BaseHinter: hint.NewBaseHinter(NumberValueHint)}
)

type NumberValue struct {
	hint.BaseHinter
	v interface{}
	b []byte
	h valuehash.Hash
	t reflect.Kind
}

func NewNumberValue(v interface{}) (NumberValue, error) {
	return NumberValue{BaseHinter: hint.NewBaseHinter(NumberValueHint)}.set(v)
}

func (nv NumberValue) set(v interface{}) (NumberValue, error) {
	var b []byte
	switch t := v.(type) {
	case int, int8, int16, int32, int64:
		var i int64
		switch it := v.(type) {
		case int:
			i = int64(it)
		case int8:
			i = int64(it)
		case int16:
			i = int64(it)
		case int32:
			i = int64(it)
		case int64:
			i = it
		}

		b = util.Int64ToBytes(i)
	case uint, uint8, uint16, uint32, uint64:
		var i uint64
		switch it := v.(type) {
		case uint:
			i = uint64(it)
		case uint8:
			i = uint64(it)
		case uint16:
			i = uint64(it)
		case uint32:
			i = uint64(it)
		case uint64:
			i = it
		}

		b = util.Uint64ToBytes(i)
	case float64:
		b = util.Float64ToBytes(t)
	default:
		return NumberValue{}, errors.Errorf("not number-like: %T", v)
	}

	return NumberValue{
		BaseHinter: nv.BaseHinter,
		v:          v,
		b:          b,
		h:          valuehash.NewSHA256(b),
		t:          reflect.TypeOf(v).Kind(),
	}, nil
}

func (nv NumberValue) IsValid([]byte) error {
	switch nv.t {
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64:
	default:
		return errors.Errorf("invalid number type: %v", nv.t)
	}

	if err := isvalid.Check([]isvalid.IsValider{nv.BaseHinter, nv.h}, nil, false); err != nil {
		return err
	}

	if nv.b == nil || len(nv.b) < 1 {
		return errors.Errorf("empty bytes for NumberValue")
	}

	return nil
}

func (nv NumberValue) Equal(v Value) bool {
	return nv.Hash().Equal(v.Hash())
}

func (nv NumberValue) Hash() valuehash.Hash {
	return nv.h
}

func (nv NumberValue) Interface() interface{} {
	return nv.v
}

func (nv NumberValue) Set(v interface{}) (Value, error) {
	return nv.set(v)
}

func (nv NumberValue) Type() reflect.Kind {
	return nv.t
}
