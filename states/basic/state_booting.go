package basicstates

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type BootingState struct {
	*logging.Logging
	*BaseState
	database  storage.Database
	blockData blockdata.BlockData
	policy    *isaac.LocalPolicy
	suffrage  base.Suffrage
}

func NewBootingState(
	st storage.Database,
	blockData blockdata.BlockData,
	policy *isaac.LocalPolicy,
	suffrage base.Suffrage,
) *BootingState {
	return &BootingState{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "basic-booting-state")
		}),
		BaseState: NewBaseState(base.StateBooting),
		database:  st,
		blockData: blockData,
		policy:    policy,
		suffrage:  suffrage,
	}
}

func (st *BootingState) Enter(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Enter(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	if _, err := storage.CheckBlock(st.database, st.policy.NetworkID()); err != nil {
		st.Log().Error().Err(err).Msg("something wrong to check blocks")

		if !xerrors.Is(err, util.NotFoundError) {
			return nil, err
		}

		st.Log().Debug().Msg("empty blocks found; cleaning up")
		// NOTE empty block
		if err := blockdata.Clean(st.database, st.blockData, false); err != nil {
			return nil, err
		}

		if len(st.suffrage.Nodes()) < 2 { // NOTE suffrage nodes has local node itself
			st.Log().Debug().Msg("empty blocks; no other nodes in suffrage; can not sync")

			return nil, xerrors.Errorf("empty blocks, but no other nodes; can not sync")
		} else {
			st.Log().Debug().Msg("empty blocks; will sync")

			return func() error {
				if err := callback(); err != nil {
					return err
				}

				return NewStateSwitchContext(base.StateBooting, base.StateSyncing)
			}, nil
		}
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		st.Log().Debug().Msg("block checked; moves to joining")

		return NewStateSwitchContext(base.StateBooting, base.StateJoining)
	}, nil
}
