package encoder

import (
	"reflect"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

func Ptr(i interface{}) (reflect.Value /* ptr */, reflect.Value /* elem */) {
	elem := reflect.ValueOf(i)
	if elem.Type().Kind() == reflect.Ptr {
		return elem, elem.Elem()
	}

	if elem.CanAddr() {
		return elem.Addr(), elem
	}

	ptr := reflect.New(elem.Type())
	ptr.Elem().Set(elem)

	return ptr, elem
}

type (
	UnpackFunc func([]byte, hint.Hint) (interface{}, error)
	Unpacker   struct {
		Elem interface{}
		N    string
		F    UnpackFunc
	}
)

func AnalyzeSetHinter(up Unpacker) Unpacker {
	elem := up.Elem
	if _, ok := elem.(hint.SetHinter); !ok {
		return up
	}

	oht := elem.(hint.Hinter).Hint()

	// NOTE hint.BaseHinter
	var found bool
	if i, j := reflect.TypeOf(elem).FieldByName("BaseHinter"); j && i.Type == reflect.TypeOf(hint.BaseHinter{}) {
		found = true
	}

	if !found {
		p := up.F
		up.F = func(b []byte, ht hint.Hint) (interface{}, error) {
			i, err := p(b, ht)
			if err != nil {
				return i, err
			}

			if ht.IsZero() {
				ht = oht
			}

			return i.(hint.SetHinter).SetHint(ht), nil
		}

		return up
	}

	p := up.F
	up.F = func(b []byte, ht hint.Hint) (interface{}, error) {
		i, err := p(b, ht)
		if err != nil {
			return i, err
		}

		n := reflect.New(reflect.TypeOf(i))
		n.Elem().Set(reflect.ValueOf(i))

		v := n.Elem().FieldByName("BaseHinter")
		if !v.IsValid() || !v.CanAddr() {
			return i, nil
		}

		if ht.IsZero() {
			ht = oht
		}

		v.Set(reflect.ValueOf(hint.NewBaseHinter(ht)))

		return n.Elem().Interface(), nil
	}

	return up
}

func Decode(b []byte, enc Encoder, target interface{}) error {
	if len(b) < 1 {
		return nil
	}

	switch i, err := enc.Decode(b); {
	case err != nil:
		return err
	case i == nil:
		return nil
	default:
		return util.InterfaceSetValue(i, target)
	}
}

func DecodeWithHint(b []byte, enc Encoder, ht hint.Hint, target interface{}) error {
	if len(b) < 1 {
		return nil
	}

	switch i, err := enc.DecodeWithHint(b, ht); {
	case err != nil:
		return err
	case i == nil:
		return nil
	default:
		return util.InterfaceSetValue(i, target)
	}
}
