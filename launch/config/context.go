package config

import (
	"context"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	ContextValueConfig      util.ContextKey = "config"
	ContextValueEncoders    util.ContextKey = "encoders"
	ContextValueJSONEncoder util.ContextKey = "json_encoder"
	ContextValueBSONEncoder util.ContextKey = "bson_encoder"
	ContextValueLog         util.ContextKey = "log"
	ContextValueNetworkLog  util.ContextKey = "network_log"
)

func LoadConfigContextValue(ctx context.Context, l *LocalNode) error {
	return util.LoadFromContextValue(ctx, ContextValueConfig, l)
}

func LoadEncodersContextValue(ctx context.Context, l **encoder.Encoders) error {
	return util.LoadFromContextValue(ctx, ContextValueEncoders, l)
}

func LoadJSONEncoderContextValue(ctx context.Context, l **jsonenc.Encoder) error {
	return util.LoadFromContextValue(ctx, ContextValueJSONEncoder, l)
}

func LoadBSONEncoderContextValue(ctx context.Context, l **bsonenc.Encoder) error {
	return util.LoadFromContextValue(ctx, ContextValueBSONEncoder, l)
}

func LoadLogContextValue(ctx context.Context, l *logging.Logger) error {
	return util.LoadFromContextValue(ctx, ContextValueLog, l)
}

func LoadNetworkLogContextValue(ctx context.Context, l *logging.Logger) error {
	return util.LoadFromContextValue(ctx, ContextValueNetworkLog, l)
}
