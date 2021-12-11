package util

import (
	"reflect"

	"github.com/pkg/errors"
)

func InterfaceSetValue(v, target interface{}) error {
	switch {
	case v == nil:
		return nil
	case target == nil:
		return errors.Errorf("target should be not nil")
	}

	value := reflect.ValueOf(target)
	if value.Type().Kind() != reflect.Ptr {
		return errors.Errorf("target should be pointer")
	}

	elem := value.Elem()

	switch t := elem.Type(); t.Kind() {
	case reflect.Interface:
		if !reflect.TypeOf(v).Implements(t) {
			return errors.Errorf("%T not implements the target, %T", v, elem.Interface())
		}
	default:
		if elem.Type() != reflect.TypeOf(v) {
			return errors.Errorf("%T, not the expected %T type", v, elem.Interface())
		} else if !elem.CanSet() {
			return errors.Errorf("target can not set")
		}
	}

	elem.Set(reflect.ValueOf(v))

	return nil
}
