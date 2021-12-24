package deploy

import (
	"context"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util/logging"
)

var HookNameBlockdataCleaner = "blockdata_cleaner"

func HookBlockdataCleaner(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var lbd *localfs.Blockdata
	var bd blockdata.Blockdata
	if err := process.LoadBlockdataContextValue(ctx, &bd); err != nil {
		return ctx, err
	} else if i, ok := bd.(*localfs.Blockdata); !ok {
		return ctx, errors.Errorf("to clean blockdata, needs localfs.Blockdata, not %T", bd)
	} else {
		lbd = i
	}

	bc := NewBlockdataCleaner(lbd, DefaultTimeAfterRemoveBlockdataFiles)
	_ = bc.SetLogging(log)

	if err := bc.Start(); err != nil {
		return ctx, err
	}

	log.Log().Debug().Dur("remove_after", DefaultTimeAfterRemoveBlockdataFiles).Msg("BlockdataCleaner created")

	return context.WithValue(ctx, ContextValueBlockdataCleaner, bc), nil
}
