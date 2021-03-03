package util

import (
	"context"
	"reflect"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/errors"
)

var ContextValueNotFoundError = errors.NewError("not found in context")

type ContextKey string

func LoadFromContextValue(ctx context.Context, key ContextKey, t interface{}) error {
	var cv interface{}
	if i := ctx.Value(key); i == nil {
		return ContextValueNotFoundError.Errorf(string(key))
	} else {
		cv = i
	}

	value := reflect.ValueOf(t)
	elem := value.Elem().Interface()

	switch ty := reflect.TypeOf(t).Elem(); ty.Kind() {
	case reflect.Interface:
		if !reflect.TypeOf(cv).Implements(ty) {
			return xerrors.Errorf("%T not implements the target, %T", cv, elem)
		}
	default:
		if reflect.TypeOf(elem) != reflect.TypeOf(cv) {
			return xerrors.Errorf("%T, not the expected %T type in context", cv, elem)
		} else if !value.Elem().CanSet() {
			return xerrors.Errorf("target can not set")
		}
	}

	value.Elem().Set(reflect.ValueOf(cv))

	return nil
}
