package process

import (
	"context"
	"time"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

const ProcessNameTimeSyncer = "time-syncer"

var ProcessorTimeSyncer pm.Process

func init() {
	if i, err := pm.NewProcess(
		ProcessNameTimeSyncer,
		[]string{
			ProcessNameConfig,
		},
		ProcessTimeSyncer,
	); err != nil {
		panic(err)
	} else {
		ProcessorTimeSyncer = i
	}
}

func ProcessTimeSyncer(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var conf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
		return ctx, err
	}

	if len(conf.LocalConfig().TimeServer()) < 1 {
		log.Debug().Msg("no time server; local time will be used")

		return ctx, nil
	}

	ts, err := localtime.NewTimeSyncer(conf.LocalConfig().TimeServer(), time.Minute*2)
	if err != nil {
		return ctx, err
	}
	_ = ts.SetLogger(log)

	if err := ts.Start(); err != nil {
		return ctx, err
	}

	localtime.SetTimeSyncer(ts)

	return ctx, nil
}
