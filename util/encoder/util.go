package encoder

import (
	"reflect"
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
