package encoder

import (
	"reflect"

	"github.com/spikeekips/mitum/util/hint"
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
		if err := encs.AddHinter(hinters[i]); err != nil {
			return nil, err
		}
	}

	return encs, nil
}
