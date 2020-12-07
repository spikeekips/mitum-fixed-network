package process

import (
	"context"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/util/logging"
)

func HookConfigVerbose(ctx context.Context) (context.Context, error) {
	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return ctx, err
	}

	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	log.Debug().Interface("config", conf).Msg("config loaded")

	return ctx, nil
}
