package basicstates

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

type syncBlockEvent struct {
	voteproof base.Voteproof
	height    base.Height
	errch     chan error
}

func newSyncBlockEvent() syncBlockEvent {
	return syncBlockEvent{
		voteproof: nil,
		height:    base.NilHeight,
		errch:     make(chan error),
	}
}

func (e syncBlockEvent) setVoteproof(voteproof base.Voteproof) syncBlockEvent {
	e.voteproof = voteproof

	return e
}

func (e syncBlockEvent) setHeight(height base.Height) syncBlockEvent {
	e.height = height

	return e
}

type SyncingState struct {
	sync.RWMutex
	*logging.Logging
	*BaseState
	database             storage.Database
	blockData            blockdata.BlockData
	policy               *isaac.LocalPolicy
	nodepool             *network.Nodepool
	suffrage             base.Suffrage
	syncs                *isaac.Syncers
	waitVoteproofTimeout time.Duration
	nc                   *network.NodeInfoChecker
	notifyNewBlockCancel func()
	newBlockEventch      chan syncBlockEvent
}

func NewSyncingState(
	db storage.Database,
	blockData blockdata.BlockData,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
) *SyncingState {
	return &SyncingState{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "basic-syncing-state")
		}),
		BaseState:            NewBaseState(base.StateSyncing),
		database:             db,
		blockData:            blockData,
		policy:               policy,
		nodepool:             nodepool,
		suffrage:             suffrage,
		waitVoteproofTimeout: time.Second * 5, // NOTE long enough time
		newBlockEventch:      make(chan syncBlockEvent),
	}
}

func (st *SyncingState) Enter(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Enter(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	if st.syncers() != nil {
		return nil, errors.Errorf("not stopped correctly; syncers still running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	st.notifyNewBlockCancel = cancel

	go func() {
	end:
		for {
			select {
			case <-ctx.Done():
				break end
			case event := <-st.newBlockEventch:
				switch {
				case st.syncers() == nil:
					continue
				case st.exiting.Value().(bool):
					continue
				}

				switch {
				case event.voteproof != nil:
					event.errch <- st.processVoteproof(event.voteproof)
				case event.height > base.NilHeight:
					event.errch <- st.whenNewHeight(event.height)
				}
			}
		}
	}()

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.enterCallback(sctx.Voteproof())
	}, nil
}

func (st *SyncingState) Exit(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Exit(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	if err := st.stopNewBlockEvent(); err != nil {
		return nil, err
	}

	if st.notifyNewBlockCancel != nil {
		st.notifyNewBlockCancel()
		st.notifyNewBlockCancel = nil
	}

	return callback, nil
}

func (st *SyncingState) ProcessVoteproof(voteproof base.Voteproof) error {
	return st.newBlockEvent(newSyncBlockEvent().setVoteproof(voteproof))
}

func (st *SyncingState) processVoteproof(voteproof base.Voteproof) error {
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

	var syncableChannels func() map[string]network.Channel
	if st.States != nil {
		syncableChannels = st.BaseState.syncableChannels
	} else {
		syncableChannels = st.syncableChannelsOfNodepool
	}

	syncs := isaac.NewSyncers(st.database, st.blockData, st.policy, baseManifest, syncableChannels)
	syncs.WhenBlockSaved(st.whenBlockSaved)
	syncs.WhenFinished(st.whenFinished)

	_ = syncs.SetLogging(st.Logging)

	if err := syncs.Start(); err != nil {
		return err
	}
	st.setSyncers(syncs)

	if st.canStartNodeInfoChecker() {
		st.Log().Debug().Msg("local is not suffrage node, NodeInfoChecker started")

		st.nc = network.NewNodeInfoChecker(
			st.policy.NetworkID(),
			st.nodepool,
			0,
			func(height base.Height) error {
				return st.newBlockEvent(newSyncBlockEvent().setHeight(height))
			},
		)
		_ = st.nc.SetLogging(st.Logging)
		if err := st.nc.Start(); err != nil {
			return err
		}
	}

	if voteproof != nil {
		l := st.Log().With().Str("voteproof_id", voteproof.ID()).Logger()
		if baseManifest != nil {
			l.Debug().Int64("base_height", baseManifest.Height().Int64())
		}

		l.Debug().Msg("new syncers started with voteproof")

		return st.newBlockEvent(newSyncBlockEvent().setVoteproof(voteproof))
	}
	e := st.Log().Debug()
	if baseManifest != nil {
		e.Int64("base_height", baseManifest.Height().Int64())
	}

	e.Msg("new syncers started without voteproof")

	return nil
}

func (st *SyncingState) syncers() *isaac.Syncers {
	st.RLock()
	defer st.RUnlock()

	return st.syncs
}

func (st *SyncingState) setSyncers(syncs *isaac.Syncers) {
	st.Lock()
	defer st.Unlock()

	st.syncs = syncs
}

func (st *SyncingState) whenBlockSaved(blks []block.Block) {
	if len(blks) < 1 {
		panic("empty saved blocks in SyncingStateNoneSuffrage")
	}

	sort.Slice(blks, func(i, j int) bool {
		return blks[i].Height()-blks[j].Height() < 0
	})

	ivp := blks[len(blks)-1].ConsensusInfo().INITVoteproof()
	_ = st.SetLastVoteproof(ivp)

	if err := st.NewBlocks(blks); err != nil {
		st.Log().Error().Err(err).Msg("new blocks hooks failed")
	}
}

func (st *SyncingState) whenFinished(height base.Height) {
	l := st.Log().With().Int64("height", height.Int64()).Logger()

	voteproof := st.database.LastVoteproof(base.StageACCEPT)
	_ = st.SetLastVoteproof(voteproof)

	if st.suffrage.IsInside(st.nodepool.LocalNode().Address()) {
		l.Debug().Msg("syncing finished; will wait new voteproof")

		if err := st.waitVoteproof(); err != nil {
			l.Error().Err(err).Stringer("timer", TimerIDSyncingWaitVoteproof).Msg("failed to start timer")

			return
		}
	}
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
		return errors.Errorf("invalid Voteproof received")
	}

	switch {
	case baseHeight > to:
		l.Debug().Msg("voteproof has lower height")

		return nil
	case baseHeight < to:
		return st.syncFromVoteproof(voteproof, to)
	default:
		if !st.syncers().IsFinished() {
			l.Debug().Msg("expected init voteproof received, but not finished")

			return nil
		}

		if st.canMoveConsensus() {
			l.Debug().Msg("init voteproof, expected; moves to consensus")

			_ = st.SetLastVoteproof(voteproof)

			if err := st.stopNewBlockEvent(); err != nil {
				return err
			}

			_ = st.exiting.Set(true)
			return st.NewStateSwitchContext(base.StateConsensus).SetVoteproof(voteproof)
		}

		l.Debug().Msg("expected init voteproof received, but will stay in syncing")

		return nil
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
		if n, _, found := st.nodepool.Node(nf.FactSign().Node()); !found {
			return errors.Errorf("node, %q in voteproof is not known node", nf.FactSign().Node())
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

func (st *SyncingState) waitVoteproof() error {
	if st.Timers().IsTimerStarted(TimerIDSyncingWaitVoteproof) {
		return nil
	}

	timer := localtime.NewContextTimer(
		TimerIDSyncingWaitVoteproof,
		0,
		func(int) (bool, error) {
			if !st.canMoveConsensus() {
				return true, nil
			}

			if syncs := st.syncers(); syncs != nil {
				if !syncs.IsFinished() {
					st.Log().Debug().Msg("syncer is still running; timer will be stopped")

					return true, nil
				}
			}

			st.Log().Debug().Msg("syncing finished, but no more Voteproof; moves to joining state")

			if err := st.StateSwitch(st.NewStateSwitchContext(base.StateJoining)); err != nil {
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

func (st *SyncingState) whenNewHeight(height base.Height) error {
	st.Lock()
	defer st.Unlock()

	if st.syncs == nil {
		return nil
	}

	if lvp := st.LastVoteproof(); lvp != nil && height <= lvp.Height() {
		return nil
	}

	n := st.nodepool.LenRemoteAlives()
	if n < 1 {
		return nil
	}

	sources := make([]base.Node, n)

	var i int
	st.nodepool.TraverseAliveRemotes(func(no base.Node, _ network.Channel) bool {
		sources[i] = no
		i++

		return true
	})

	if _, err := st.syncs.Add(height, sources); err != nil {
		st.Log().Error().Err(err).Int64("height", height.Int64()).Msg("failed to add syncers")

		return err
	}

	return nil
}

func (st *SyncingState) canStartNodeInfoChecker() bool {
	switch {
	case !st.suffrage.IsInside(st.nodepool.LocalNode().Address()):
		return true
	case st.underHandover():
		return true
	default:
		return false
	}
}

func (st *SyncingState) canMoveConsensus() bool {
	switch {
	case !st.suffrage.IsInside(st.nodepool.LocalNode().Address()):
		st.Log().Debug().Msg("local is not in suffrage; will stay in syncing")

		return false
	case st.States == nil:
		return true
	case st.underHandover():
		if st.States.isHandoverReady() {
			st.Log().Debug().Msg("under handover and is ready; can move handover")

			return true
		}

		st.Log().Debug().Msg("under handover, but is not ready; will stay in syncing")

		return false
	default:
		return true
	}
}

func (st *SyncingState) syncableChannelsOfNodepool() map[string]network.Channel {
	pn := map[string]network.Channel{}

	st.nodepool.TraverseAliveRemotes(func(no base.Node, ch network.Channel) bool {
		pn[no.String()] = ch

		return true
	})

	return pn
}

func (st *SyncingState) newBlockEvent(event syncBlockEvent) error {
	if st.syncers() == nil {
		return nil
	}

	st.newBlockEventch <- event

	return <-event.errch
}

func (st *SyncingState) stopNewBlockEvent() error {
	if err := st.stopWaitVoteproof(); err != nil {
		return err
	}

	if syncs := st.syncers(); syncs != nil {
		if err := syncs.Stop(); err != nil {
			return err
		}
	}

	st.setSyncers(nil)

	if st.nc != nil {
		if err := st.nc.Stop(); err != nil {
			if !errors.Is(err, util.DaemonAlreadyStoppedError) {
				return err
			}
		}

		st.nc = nil
	}

	return nil
}
