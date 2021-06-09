package process

import (
	"context"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

const HookNameAddHinters = "add_hinters"

func HookAddHinters(types []hint.Type, hinters []hint.Hinter) pm.ProcessFunc {
	return func(ctx context.Context) (context.Context, error) {
		var encs *encoder.Encoders
		if err := config.LoadEncodersContextValue(ctx, &encs); err != nil {
			return ctx, err
		}

		for i := range types {
			if err := encs.AddType(types[i]); err != nil {
				return ctx, err
			}
		}

		for i := range hinters {
			if err := encs.AddHinter(hinters[i]); err != nil {
				return ctx, err
			}
		}

		if err := encs.Initialize(); err != nil {
			return ctx, err
		}

		return ctx, nil
	}
}
