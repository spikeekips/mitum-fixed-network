package util

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
)

var ContextValueNotFoundError = NewError("not found in context")

type ContextKey string

func LoadFromContextValue(ctx context.Context, key ContextKey, t interface{}) error {
	cv := ctx.Value(key)
	if cv == nil {
		return ContextValueNotFoundError.Errorf(string(key))
	}

	value := reflect.ValueOf(t)
	elem := value.Elem().Interface()

	switch ty := reflect.TypeOf(t).Elem(); ty.Kind() {
	case reflect.Interface:
		if !reflect.TypeOf(cv).Implements(ty) {
			return errors.Errorf("%T not implements the target, %T", cv, elem)
		}
	default:
		if reflect.TypeOf(elem) != reflect.TypeOf(cv) {
			return errors.Errorf("%T, not the expected %T type in context", cv, elem)
		} else if !value.Elem().CanSet() {
			return errors.Errorf("target can not set")
		}
	}

	value.Elem().Set(reflect.ValueOf(cv))

	return nil
}
