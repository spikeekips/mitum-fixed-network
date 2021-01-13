package isaac

import (
	"context"

	"github.com/spikeekips/mitum/base/prprocessor"
)

func (pp *DefaultProcessor) Save(ctx context.Context) error {
	pp.Lock()
	defer pp.Unlock()

	if err := pp.resetSave(); err != nil {
		return err
	}

	pp.setState(prprocessor.Saving)

	if err := pp.save(ctx); err != nil {
		pp.setState(prprocessor.SaveFailed)

		return err
	} else {
		pp.setState(prprocessor.Saved)

		return nil
	}
}

func (pp *DefaultProcessor) save(ctx context.Context) error {
	pp.Log().Debug().Msg("trying to save")

	if pp.preSaveHook != nil {
		if err := pp.preSaveHook(ctx); err != nil {
			return err
		}
	}

	for _, f := range []func(context.Context) error{
		pp.storeStorage,
		pp.storeBlockFS,
	} {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := f(ctx); err != nil {
				pp.Log().Error().Err(err).Msg("failed to save")
				return err
			}
		}
	}

	if pp.postSaveHook != nil {
		if err := pp.postSaveHook(ctx); err != nil {
			return err
		}
	}

	pp.Log().Debug().Msg("saved")

	return nil
}

func (pp *DefaultProcessor) storeStorage(ctx context.Context) error {
	pp.Log().Debug().Msg("trying to store storage")

	if err := pp.bs.Commit(ctx); err != nil {
		pp.Log().Error().Err(err).Msg("failed to store storage")

		return err
	} else if err := pp.bs.Close(); err != nil {
		return err
	} else {
		pp.bs = nil

		pp.Log().Debug().Msg("stored storage")

		return nil
	}
}

func (pp *DefaultProcessor) storeBlockFS(context.Context) error {
	pp.Log().Debug().Msg("trying to store BlockFS")

	if err := pp.blockFS.Commit(pp.blk.Height(), pp.blk.Hash()); err != nil {
		pp.Log().Error().Err(err).Msg("trying to store BlockFS")

		return err
	}

	pp.Log().Debug().Msg("stored BlockFS")

	return nil
}

func (pp *DefaultProcessor) resetSave() error {
	switch pp.state {
	case prprocessor.BeforePrepared,
		prprocessor.Preparing,
		prprocessor.PrepareFailed,
		prprocessor.Prepared,
		prprocessor.SaveFailed,
		prprocessor.Saved,
		prprocessor.Canceled:
		return nil
	}

	pp.Log().Debug().Str("state", pp.state.String()).Msg("save will be resetted")

	if err := pp.st.CleanByHeight(pp.proposal.Height()); err != nil {
		return err
	} else if err := pp.blockFS.CleanByHeight(pp.proposal.Height()); err != nil {
		return err
	}

	return nil
}
