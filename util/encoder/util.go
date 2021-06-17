package encoder

import (
	"reflect"
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
	UnpackFunc func([]byte) (interface{}, error)
	Unpacker   struct {
		Elem interface{}
		N    string
		F    UnpackFunc
	}
)
