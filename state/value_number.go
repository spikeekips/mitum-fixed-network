package state

import (
	"reflect"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	NumberValueType = hint.MustNewType(0x12, 0x00, "state-number-value")
	NumberValueHint = hint.MustHint(NumberValueType, "0.0.1")
)

type NumberValue struct {
	v interface{}
	b []byte
	h valuehash.Hash
	t reflect.Kind
}

func NewNumberValue(v interface{}) (NumberValue, error) {
	return NumberValue{}.set(v)
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
		return NumberValue{}, xerrors.Errorf("not number-like: %T", v)
	}

	return NumberValue{
		v: v,
		b: b,
		h: valuehash.NewSHA256(b),
		t: reflect.TypeOf(v).Kind(),
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
		return xerrors.Errorf("invalid number type: %v", nv.t)
	}

	if err := nv.h.IsValid(nil); err != nil {
		return err
	}
	if nv.b == nil || len(nv.b) < 1 {
		return xerrors.Errorf("empty bytes for NumberValue")
	}

	return nil
}

func (nv NumberValue) Bytes() []byte {
	return nv.b
}

func (nv NumberValue) Hint() hint.Hint {
	return NumberValueHint
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
