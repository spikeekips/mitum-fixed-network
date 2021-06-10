package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

const ProcessNameEncoders = "encoders"

var ProcessorEncoders pm.Process

func init() {
	if i, err := pm.NewProcess(ProcessNameEncoders, nil, ProcessEncoders); err != nil {
		panic(err)
	} else {
		ProcessorEncoders = i
	}
}

func ProcessEncoders(ctx context.Context) (context.Context, error) {
	jenc := jsonenc.NewEncoder()
	benc := bsonenc.NewEncoder()
	encs, err := encoder.LoadEncoders([]encoder.Encoder{jenc, benc})
	if err != nil {
		return ctx, xerrors.Errorf("failed to load encoders: %w", err)
	}

	ctx = context.WithValue(ctx, config.ContextValueEncoders, encs)
	ctx = context.WithValue(ctx, config.ContextValueJSONEncoder, jenc)
	ctx = context.WithValue(ctx, config.ContextValueBSONEncoder, benc)

	return ctx, nil
}
