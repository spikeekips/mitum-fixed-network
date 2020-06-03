package encoder

import (
	"reflect"

	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

func ExtractPtr(i interface{}) (reflect.Value, reflect.Value) {
	var ptr reflect.Value = reflect.ValueOf(i)
	var elem reflect.Value = ptr
	if ptr.Type().Kind() == reflect.Ptr {
		elem = ptr.Elem()
	} else {
		if elem.CanAddr() {
			ptr = elem.Addr()
		} else {
			ptr = reflect.New(elem.Type())
			ptr.Elem().Set(elem)
		}
	}

	return ptr, elem
}

func LoadEncoders(encoders []Encoder, hinters ...hint.Hinter) (*Encoders, error) {
	encs := NewEncoders()

	for _, enc := range encoders {
		if err := encs.AddEncoder(enc); err != nil {
			return nil, err
		}
	}

	for i := range hinters {
		hinter, ok := hinters[i].(hint.Hinter)
		if !ok {
			return nil, xerrors.Errorf("not hint.Hinter: %T", hinters[i])
		}

		if err := encs.AddHinter(hinter); err != nil {
			return nil, err
		}
	}

	return encs, nil
}
