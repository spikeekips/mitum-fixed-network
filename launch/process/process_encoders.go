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

	encoders := []encoder.Encoder{jenc, benc}
	encs := encoder.NewEncoders()
	for _, enc := range encoders {
		if err := encs.AddEncoder(enc); err != nil {
			return ctx, xerrors.Errorf("failed to load encoders: %w", err)
		}
	}

	ctx = context.WithValue(ctx, config.ContextValueEncoders, encs)
	ctx = context.WithValue(ctx, config.ContextValueJSONEncoder, jenc)
	ctx = context.WithValue(ctx, config.ContextValueBSONEncoder, benc)

	return ctx, nil
}
