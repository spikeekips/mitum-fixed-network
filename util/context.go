package util

import (
	"context"
)

var ContextValueNotFoundError = NewError("not found in context")

type ContextKey string

func LoadFromContextValue(ctx context.Context, key ContextKey, target interface{}) error {
	cv := ctx.Value(key)
	if cv == nil {
		return ContextValueNotFoundError.Errorf(string(key))
	}

	return InterfaceSetValue(cv, target)
}
