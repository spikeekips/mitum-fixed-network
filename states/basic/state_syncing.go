package basicstates

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type SyncingState struct {
	*BaseSyncingState
	waitVoteproofTimeout time.Duration
}

func NewSyncingState(
	db storage.Database,
	blockData blockdata.BlockData,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
) *SyncingState {
	return &SyncingState{
		BaseSyncingState:     NewBaseSyncingState("basic-syncing-state", db, blockData, policy, nodepool),
		waitVoteproofTimeout: time.Second * 5, // NOTE long enough time
	}
}

func (st *SyncingState) Enter(sctx StateSwitchContext) (func() error, error) {
	var callback func() error
	if i, err := st.BaseSyncingState.Enter(sctx); err != nil {
		return nil, err
	} else {
		callback = i
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.enterCallback(sctx.Voteproof())
	}, nil
}

func (st *SyncingState) Exit(sctx StateSwitchContext) (func() error, error) {
	var callback func() error
	if i, err := st.BaseSyncingState.Exit(sctx); err != nil {
		return nil, err
	} else {
		callback = i
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.stopWaitVoteproof()
	}, nil
}

func (st *SyncingState) ProcessVoteproof(voteproof base.Voteproof) error {
	switch voteproof.Stage() {
	case base.StageINIT:
		return st.handleINITTVoteproof(voteproof)
	case base.StageACCEPT:
		return st.handleACCEPTVoteproof(voteproof)
	default:
		return nil
	}
}

func (st *SyncingState) enterCallback(voteproof base.Voteproof) error {
	var baseManifest block.Manifest
	if m, found, err := st.database.LastManifest(); err != nil {
		return err
	} else if found {
		baseManifest = m
	}

	syncs := st.syncers()
	syncs.WhenFinished(st.whenFinished)

	if voteproof != nil {
		e := isaac.LoggerWithVoteproof(voteproof, st.Log()).Debug()
		if baseManifest != nil {
			e.Hinted("base_height", baseManifest.Height())
		}

		e.Msg("new syncers started with voteproof")

		return st.ProcessVoteproof(voteproof)
	} else {
		e := st.Log().Debug()
		if baseManifest != nil {
			e.Hinted("base_height", baseManifest.Height())
		}

		e.Msg("new syncers started without voteproof")

		return nil
	}
}

func (st *SyncingState) handleINITTVoteproof(voteproof base.Voteproof) error {
	baseHeight := base.PreGenesisHeight
	if m, found, err := st.database.LastManifest(); err != nil {
		return err
	} else if found {
		baseHeight = m.Height()
	}

	l := st.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("voteproof_stage", voteproof.Stage()).
			Hinted("voteproof_height", voteproof.Height()).
			Hinted("voteproof_round", voteproof.Round()).
			Hinted("local_height", baseHeight)
	})

	var to base.Height
	switch voteproof.Stage() {
	case base.StageINIT:
		to = voteproof.Height() - 1
	case base.StageACCEPT:
		to = voteproof.Height()
	default:
		return xerrors.Errorf("invalid Voteproof received")
	}

	switch {
	case baseHeight > to:
		l.Debug().Msg("voteproof has lower height")

		return nil
	case baseHeight < to:
		return st.syncFromVoteproof(voteproof, to)
	default:
		if !st.syncers().IsFinished() {
			return nil
		}

		l.Debug().Msg("init voteproof, expected")

		if err := st.stopWaitVoteproof(); err != nil {
			return err
		}

		l.Debug().Msg("init voteproof, expected; moves to consensus")

		return NewStateSwitchContext(base.StateSyncing, base.StateConsensus).SetVoteproof(voteproof)
	}
}

func (st *SyncingState) handleACCEPTVoteproof(voteproof base.Voteproof) error {
	baseHeight := base.PreGenesisHeight
	if m, found, err := st.database.LastManifest(); err != nil {
		return err
	} else if found {
		baseHeight = m.Height()
	}

	l := st.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("voteproof_stage", voteproof.Stage()).
			Hinted("voteproof_height", voteproof.Height()).
			Hinted("voteproof_round", voteproof.Round()).
			Hinted("local_height", baseHeight)
	})

	if baseHeight >= voteproof.Height() {
		l.Debug().Msg("voteproof has lower height")

		return nil
	}

	return st.syncFromVoteproof(voteproof, voteproof.Height())
}

func (st *SyncingState) syncFromVoteproof(voteproof base.Voteproof, to base.Height) error {
	var sourceNodes []network.Node
	for i := range voteproof.Votes() {
		nf := voteproof.Votes()[i]
		if n, found := st.nodepool.Node(nf.Node()); !found {
			return xerrors.Errorf("node, %q in voteproof is not known node", nf.Node())
		} else if !n.Address().Equal(st.nodepool.Local().Address()) {
			sourceNodes = append(sourceNodes, n)
		}
	}

	st.Log().VerboseFunc(func(e *logging.Event) logging.Emitter {
		var addresses []string
		for _, n := range sourceNodes {
			addresses = append(addresses, n.Address().String())
		}

		return e.Strs("source_nodes", addresses)
	}).
		Hinted("voteproof_height", voteproof.Height()).
		Hinted("voteproof_round", voteproof.Round()).
		Hinted("height_to", to).
		Msg("will sync to the height")

	if isFinished, err := st.syncers().Add(to, sourceNodes); !isFinished {
		if err0 := st.stopWaitVoteproof(); err0 != nil {
			if err == nil {
				return err0
			}
		}

		return err
	} else {
		return err
	}
}

func (st *SyncingState) whenFinished(height base.Height) {
	st.Log().Debug().Hinted("height", height).Msg("syncing finished; will wait new voteproof")

	if err := st.waitVoteproof(); err != nil {
		st.Log().Error().Err(err).Str("timer", TimerIDSyncingWaitVoteproof.String()).Msg("failed to start timer")

		return
	}
}

func (st *SyncingState) waitVoteproof() error {
	timer := localtime.NewContextTimer(
		TimerIDSyncingWaitVoteproof,
		0,
		func(int) (bool, error) {
			if syncs := st.syncers(); syncs != nil {
				if !syncs.IsFinished() {
					st.Log().Debug().Msg("syncer is still running; timer will be stopped")

					return true, nil
				}
			}

			st.Log().Debug().Msg("syncing finished, but no more Voteproof; moves to joining state")

			if err := st.StateSwitch(NewStateSwitchContext(base.StateSyncing, base.StateJoining)); err != nil {
				st.Log().Error().Err(err).Msg("failed to switch state; keeps trying")
			}

			return false, nil
		},
	).SetInterval(func(i int) time.Duration {
		if i < 1 {
			return st.waitVoteproofTimeout
		}

		return time.Second * 1
	})

	if err := st.Timers().SetTimer(timer); err != nil {
		return err
	}

	return st.Timers().StartTimers([]localtime.TimerID{TimerIDSyncingWaitVoteproof}, true)
}

func (st *SyncingState) stopWaitVoteproof() error {
	return st.Timers().StopTimers([]localtime.TimerID{TimerIDSyncingWaitVoteproof})
}
