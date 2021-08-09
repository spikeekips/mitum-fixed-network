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

var HookNameBlockDataCleaner = "blockdata_cleaner"

func HookBlockDataCleaner(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var lbd *localfs.BlockData
	var bd blockdata.BlockData
	if err := process.LoadBlockDataContextValue(ctx, &bd); err != nil {
		return ctx, err
	} else if i, ok := bd.(*localfs.BlockData); !ok {
		return ctx, errors.Errorf("to clean blockdata, needs localfs.BlockData, not %T", bd)
	} else {
		lbd = i
	}

	bc := NewBlockDataCleaner(lbd, DefaultTimeAfterRemoveBlockDataFiles)
	_ = bc.SetLogging(log)

	if err := bc.Start(); err != nil {
		return ctx, err
	}

	log.Log().Debug().Dur("remove_after", DefaultTimeAfterRemoveBlockDataFiles).Msg("BlockDataCleaner created")

	return context.WithValue(ctx, ContextValueBlockDataCleaner, bc), nil
}
