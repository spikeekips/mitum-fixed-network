package state

import (
	"reflect"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

var (
	SliceValueType = hint.MustNewType(0x12, 0x03, "state-slice-value")
	SliceValueHint = hint.MustHint(SliceValueType, "0.0.1")
)

// SliceValue only supports the interface{} implements hint.Hinter and
// valuehash.Hasher().
type SliceValue struct {
	v []hint.Hinter
	b []byte
	h valuehash.Hash
}

func NewSliceValue(v interface{}) (SliceValue, error) {
	return SliceValue{}.set(v)
}

func (sv SliceValue) set(v interface{}) (SliceValue, error) {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Array, reflect.Slice:
	default:
		return SliceValue{}, xerrors.Errorf("not slice-like: %T", v)
	}

	elem := reflect.ValueOf(v)
	items := make([]hint.Hinter, elem.Len())
	bs := make([][]byte, elem.Len())
	for i := 0; i < elem.Len(); i++ {
		e := elem.Index(i).Interface()
		if e == nil {
			continue
		} else if ht, ok := e.(hint.Hinter); !ok {
			return SliceValue{}, xerrors.Errorf("item not hint.Hinter: %T", e)
		} else if _, ok := e.(valuehash.Hasher); !ok {
			return SliceValue{}, xerrors.Errorf("item not util.Byter: %T", e)
		} else {
			items[i] = ht
			bs[i] = ht.(valuehash.Hasher).Hash().Bytes()
		}
	}

	b := util.ConcatBytesSlice(bs...)

	return SliceValue{
		v: items,
		b: b,
		h: valuehash.NewSHA256(b),
	}, nil
}

func (sv SliceValue) IsValid([]byte) error {
	if err := sv.h.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (sv SliceValue) Bytes() []byte {
	return sv.b
}

func (sv SliceValue) Hint() hint.Hint {
	return SliceValueHint
}

func (sv SliceValue) Equal(v Value) bool {
	return sv.Hash().Equal(v.Hash())
}

func (sv SliceValue) Hash() valuehash.Hash {
	return sv.h
}

func (sv SliceValue) Interface() interface{} {
	return sv.v
}

func (sv SliceValue) Set(v interface{}) (Value, error) {
	return sv.set(v)
}
