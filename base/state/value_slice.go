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
	SliceValueType   = hint.Type("state-slice-value")
	SliceValueHint   = hint.NewHint(SliceValueType, "v0.0.1")
	SliceValueHinter = SliceValue{BaseHinter: hint.NewBaseHinter(SliceValueHint)}
)

// SliceValue only supports the interface{} implements hint.Hinter and
// valuehash.Hasher().
type SliceValue struct {
	hint.BaseHinter
	v []hint.Hinter
	b []byte
	h valuehash.Hash
}

func NewSliceValue(v interface{}) (SliceValue, error) {
	return SliceValue{BaseHinter: hint.NewBaseHinter(SliceValueHint)}.set(v)
}

func (sv SliceValue) set(v interface{}) (SliceValue, error) {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Array, reflect.Slice:
	default:
		return SliceValue{}, errors.Errorf("not slice-like: %T", v)
	}

	elem := reflect.ValueOf(v)
	items := make([]hint.Hinter, elem.Len())
	bs := make([][]byte, elem.Len())
	for i := 0; i < elem.Len(); i++ {
		e := elem.Index(i).Interface()
		if e == nil {
			continue
		} else if ht, ok := e.(hint.Hinter); !ok {
			return SliceValue{}, util.WrongTypeError.Errorf("item not Hinter: %T", e)
		} else if _, ok := e.(valuehash.Hasher); !ok {
			return SliceValue{}, util.WrongTypeError.Errorf("item not valuehash.Hasher: %T", e)
		} else {
			items[i] = ht
			bs[i] = ht.(valuehash.Hasher).Hash().Bytes()
		}
	}

	b := util.ConcatBytesSlice(bs...)

	return SliceValue{
		BaseHinter: sv.BaseHinter,
		v:          items,
		b:          b,
		h:          valuehash.NewSHA256(b),
	}, nil
}

func (sv SliceValue) IsValid([]byte) error {
	return isvalid.Check(nil, false, sv.BaseHinter, sv.h)
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
