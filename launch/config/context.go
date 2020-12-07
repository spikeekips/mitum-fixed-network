package config

import (
	"context"
	"reflect"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/logging"
)

var ContextValueNotFoundError = errors.NewError("not found in context")

type ContextKey string

var (
	ContextValueConfig      ContextKey = "config"
	ContextValueEncoders    ContextKey = "encoders"
	ContextValueJSONEncoder ContextKey = "json_encoder"
	ContextValueBSONEncoder ContextKey = "bson_encoder"
	ContextValueLog         ContextKey = "log"
)

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

func LoadConfigContextValue(ctx context.Context, l *LocalNode) error {
	return LoadFromContextValue(ctx, ContextValueConfig, l)
}

func LoadEncodersContextValue(ctx context.Context, l **encoder.Encoders) error {
	return LoadFromContextValue(ctx, ContextValueEncoders, l)
}

func LoadJSONEncoderContextValue(ctx context.Context, l **jsonenc.Encoder) error {
	return LoadFromContextValue(ctx, ContextValueJSONEncoder, l)
}

func LoadBSONEncoderContextValue(ctx context.Context, l **bsonenc.Encoder) error {
	return LoadFromContextValue(ctx, ContextValueBSONEncoder, l)
}

func LoadLogContextValue(ctx context.Context, l *logging.Logger) error {
	return LoadFromContextValue(ctx, ContextValueLog, l)
}
