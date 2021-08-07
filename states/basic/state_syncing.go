package basicstates

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util/localtime"
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
	callback, err := st.BaseSyncingState.Enter(sctx)
	if err != nil {
		return nil, err
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.enterCallback(sctx.Voteproof())
	}, nil
}

func (st *SyncingState) Exit(sctx StateSwitchContext) (func() error, error) {
	callback, err := st.BaseSyncingState.Exit(sctx)
	if err != nil {
		return nil, err
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
		l := st.Log().With().Str("voteproof_id", voteproof.ID()).Logger()
		if baseManifest != nil {
			l.Debug().Int64("base_height", baseManifest.Height().Int64())
		}

		l.Debug().Msg("new syncers started with voteproof")

		return st.ProcessVoteproof(voteproof)
	}
	e := st.Log().Debug()
	if baseManifest != nil {
		e.Int64("base_height", baseManifest.Height().Int64())
	}

	e.Msg("new syncers started without voteproof")

	return nil
}

func (st *SyncingState) handleINITTVoteproof(voteproof base.Voteproof) error {
	baseHeight := base.PreGenesisHeight
	if m, found, err := st.database.LastManifest(); err != nil {
		return err
	} else if found {
		baseHeight = m.Height()
	}

	l := st.Log().With().Stringer("voteproof_stage", voteproof.Stage()).
		Int64("voteproof_height", voteproof.Height().Int64()).
		Uint64("voteproof_round", voteproof.Round().Uint64()).
		Int64("local_height", baseHeight.Int64()).
		Logger()

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

	l := st.Log().With().Stringer("voteproof_stage", voteproof.Stage()).
		Int64("voteproof_height", voteproof.Height().Int64()).
		Uint64("voteproof_round", voteproof.Round().Uint64()).
		Int64("local_height", baseHeight.Int64()).
		Logger()

	if baseHeight >= voteproof.Height() {
		l.Debug().Msg("voteproof has lower height")

		return nil
	}

	return st.syncFromVoteproof(voteproof, voteproof.Height())
}

func (st *SyncingState) syncFromVoteproof(voteproof base.Voteproof, to base.Height) error {
	var sourceNodes []base.Node
	for i := range voteproof.Votes() {
		nf := voteproof.Votes()[i]
		if n, _, found := st.nodepool.Node(nf.Node()); !found {
			return xerrors.Errorf("node, %q in voteproof is not known node", nf.Node())
		} else if !n.Address().Equal(st.nodepool.LocalNode().Address()) {
			sourceNodes = append(sourceNodes, n)
		}
	}

	st.Log().Trace().Func(func(e *zerolog.Event) {
		var addresses []string
		for _, n := range sourceNodes {
			addresses = append(addresses, n.Address().String())
		}

		e.Strs("source_nodes", addresses)
	}).
		Int64("voteproof_height", voteproof.Height().Int64()).
		Uint64("voteproof_round", voteproof.Round().Uint64()).
		Int64("height_to", to.Int64()).
		Msg("will sync to the height")

	isFinished, err := st.syncers().Add(to, sourceNodes)
	if !isFinished {
		if err = st.stopWaitVoteproof(); err != nil {
			return err
		}

		return err
	}

	return err
}

func (st *SyncingState) whenFinished(height base.Height) {
	l := st.Log().With().Int64("height", height.Int64()).Logger()

	voteproof := st.database.LastVoteproof(base.StageACCEPT)
	_ = st.SetLastVoteproof(voteproof)

	l.Debug().Msg("syncing finished; will wait new voteproof")

	if err := st.waitVoteproof(); err != nil {
		l.Error().Err(err).Stringer("timer", TimerIDSyncingWaitVoteproof).Msg("failed to start timer")

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
