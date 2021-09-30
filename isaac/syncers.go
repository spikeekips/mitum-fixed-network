package isaac

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type Syncers struct {
	sync.RWMutex
	*util.ContextDaemon
	*logging.Logging
	database             storage.Database
	blockData            blockdata.BlockData
	sourceChannelsFunc   func() map[string]network.Channel
	policy               *LocalPolicy
	baseManifest         block.Manifest
	limitBlocksPerSyncer uint
	stateChan            chan SyncerStateChangedContext
	whenFinished         func(base.Height)
	whenBlockSaved       func([]block.Block)
	targetHeight         base.Height
	lastSyncer           Syncer
	sourceNodes          []base.Node
	syncers              *sync.Map
}

func NewSyncers(
	db storage.Database,
	blockData blockdata.BlockData,
	policy *LocalPolicy,
	baseManifest block.Manifest,
	sourceChannelsFunc func() map[string]network.Channel,
) *Syncers {
	sy := &Syncers{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "syncers")
		}),
		database:             db,
		blockData:            blockData,
		sourceChannelsFunc:   sourceChannelsFunc,
		policy:               policy,
		baseManifest:         baseManifest,
		limitBlocksPerSyncer: 10,
		stateChan:            make(chan SyncerStateChangedContext, 10),
		whenFinished:         func(base.Height) {},
		whenBlockSaved:       func([]block.Block) {},
		targetHeight:         base.NilHeight,
		syncers:              &sync.Map{},
	}

	sy.ContextDaemon = util.NewContextDaemon("syncers", sy.start)

	return sy
}

func (sy *Syncers) Stop() error {
	if err := sy.ContextDaemon.Stop(); err != nil {
		if !errors.Is(err, util.DaemonAlreadyStoppedError) {
			return err
		}
	}

	sy.Lock()
	defer sy.Unlock()

	sy.lastSyncer = nil

	var err error
	sy.syncers.Range(func(k, v interface{}) bool {
		err = v.(Syncer).Close()
		return err == nil
	})

	return err
}

func (sy *Syncers) SetLogging(l *logging.Logging) *logging.Logging {
	_ = sy.ContextDaemon.SetLogging(l)

	return sy.Logging.SetLogging(l)
}

func (sy *Syncers) WhenFinished(callback func(base.Height)) {
	sy.whenFinished = callback
}

func (sy *Syncers) WhenBlockSaved(callback func([]block.Block)) {
	sy.whenBlockSaved = callback
}

// Add adds new syncer with target height. If it returns true, it means Syncers
// not yet finished.
func (sy *Syncers) Add(to base.Height, sourceNodes []base.Node) (bool, error) {
	sy.Lock()
	defer sy.Unlock()

	isFinished := sy.isFinished()

	if to <= sy.targetHeight {
		return isFinished, util.IgnoreError.Errorf("lower height")
	}

	l := sy.Log().With().
		Int64("previous_height", sy.targetHeight.Int64()).Int64("new_height", to.Int64()).
		Logger()

	sy.targetHeight = to
	sy.mergeSourceNodes(sourceNodes)

	if !isFinished {
		l.Debug().Msg("target height updated")

		return false, nil
	}

	var baseManifest block.Manifest
	if sy.lastSyncer == nil {
		baseManifest = sy.baseManifest
	} else {
		baseManifest = sy.lastSyncer.TailManifest()
	}

	i, err := sy.newSyncer(baseManifest)
	if err != nil {
		l.Debug().Msg("target height updated, but failed to add new syncer")

		return false, err
	}

	sy.lastSyncer = i

	l.Debug().Msg("target height updated and new syncer added")

	go func() {
		sy.stateChan <- NewSyncerStateChangedContext(i, SyncerCreated, nil)
	}()

	return false, nil
}

func (sy *Syncers) IsFinished() bool {
	sy.RLock()
	defer sy.RUnlock()

	return sy.isFinished()
}

func (sy *Syncers) start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case sctx := <-sy.stateChan:
			if err := sy.stateChanged(sctx); err != nil {
				sy.Log().Error().Err(err).Msg("failed to handle state changed")
			}
		}
	}
}

func (sy *Syncers) newSyncer(baseManifest block.Manifest) (Syncer, error) {
	var from base.Height
	if baseManifest == nil {
		from = base.PreGenesisHeight
	} else {
		from = baseManifest.Height() + 1
	}

	if from > sy.targetHeight {
		return nil, util.IgnoreError.Errorf("already reached to target height")
	}

	to := from + base.Height(int64(sy.limitBlocksPerSyncer))
	if to > sy.targetHeight {
		to = sy.targetHeight
	}

	var l zerolog.Logger
	{
		var from base.Height
		if baseManifest == nil {
			from = base.PreGenesisHeight
		} else {
			from = baseManifest.Height() + 1
		}

		l = sy.Log().With().Int64("from", from.Int64()).Int64("to", to.Int64()).Logger()
	}

	syncer, err := NewGeneralSyncer(
		sy.database,
		sy.blockData,
		sy.policy,
		sy.sourceChannelsFunc,
		baseManifest, to,
	)
	if err != nil {
		l.Debug().Msg("failed to add new syncer")

		return nil, err
	}
	syncer = syncer.SetStateChan(sy.stateChan)

	if l, ok := (interface{})(syncer).(logging.SetLogging); ok {
		_ = l.SetLogging(sy.Logging)
	}

	l.Debug().Msg("new syncer added")

	sy.syncers.Store(syncer.ID(), syncer)

	return syncer, nil
}

func (sy *Syncers) prepareSyncer(baseManifest block.Manifest) error {
	var l zerolog.Logger
	{
		var from base.Height
		if baseManifest == nil {
			from = base.PreGenesisHeight
		} else {
			from = baseManifest.Height() + 1
		}

		l = sy.Log().With().Int64("from", from.Int64()).Logger()
	}

	newSyncer, err := sy.newSyncer(baseManifest)
	if err != nil {
		l.Debug().Err(err).Msg("failed to make new syncer")

		if errors.Is(err, util.IgnoreError) {
			return nil
		}

		return err
	}

	if err := newSyncer.Prepare(); err != nil {
		l.Error().Err(err).Msg("failed to prepare syncer")

		return err
	}

	sy.lastSyncer = newSyncer

	l.Debug().Msg("new syncer will prepare")

	return nil
}

func (sy *Syncers) stateChanged(ctx SyncerStateChangedContext) error {
	sy.Lock()
	defer sy.Unlock()

	syncer := ctx.Syncer()
	l := sy.Log().With().
		Str("syncer", syncer.ID()).
		Stringer("state", ctx.State()).
		Int64("from", syncer.HeightFrom().Int64()).
		Int64("to", syncer.HeightTo().Int64()).
		Logger()

	switch ctx.State() {
	case SyncerCreated:
		if err := sy.stateChangedCreated(ctx); err != nil {
			return err
		}
	case SyncerPrepared:
		if err := sy.stateChangedPrepared(ctx); err != nil {
			return err
		}
	case SyncerSaved:
		if err := sy.stateChangedSaved(ctx); err != nil {
			return err
		}
		if err := ctx.syncer.Close(); err != nil {
			return err
		}

		sy.syncers.Delete(ctx.syncer.ID())
	default:
		l.Debug().Msg("syncer state changed")
	}

	return nil
}

func (sy *Syncers) stateChangedCreated(ctx SyncerStateChangedContext) error {
	syncer := ctx.Syncer()

	l := sy.Log().With().
		Str("syncer", syncer.ID()).
		Stringer("state", ctx.State()).
		Int64("from", syncer.HeightFrom().Int64()).
		Int64("to", syncer.HeightTo().Int64()).
		Logger()

	if err := syncer.Prepare(); err != nil {
		l.Error().Err(err).Msg("failed to prepare")

		return err
	}
	l.Debug().Msg("new syncer will prepare")

	return nil
}

func (sy *Syncers) stateChangedPrepared(ctx SyncerStateChangedContext) error {
	syncer := ctx.Syncer()

	l := sy.Log().With().
		Str("syncer", syncer.ID()).
		Stringer("state", ctx.State()).
		Int64("from", syncer.HeightFrom().Int64()).
		Int64("to", syncer.HeightTo().Int64()).
		Logger()

	l.Debug().Msg("syncer prepared")

	return sy.prepareSyncer(syncer.TailManifest())
}

func (sy *Syncers) stateChangedSaved(ctx SyncerStateChangedContext) error {
	syncer := ctx.Syncer()

	l := sy.Log().With().
		Str("syncer", syncer.ID()).
		Stringer("state", ctx.State()).
		Int64("from", syncer.HeightFrom().Int64()).
		Int64("to", syncer.HeightTo().Int64()).
		Int64("target_height", sy.targetHeight.Int64()).
		Int64("last_syncer_height", sy.lastSyncer.HeightTo().Int64()).
		Logger()

	if db, ok := sy.database.(storage.LastBlockSaver); ok {
		if err := db.SaveLastBlock(syncer.HeightTo()); err != nil {
			return err
		}
	}

	sy.whenBlockSaved(ctx.Blocks())

	if sy.isFinished() {
		l.Debug().Msg("syncer saved and all syncers is finished")

		sy.whenFinished(syncer.HeightTo())

		return nil
	}

	if sy.lastSyncer.HeightTo() != syncer.HeightTo() {
		l.Debug().Msg("syncer saved")
	} else {
		if err := sy.prepareSyncer(syncer.TailManifest()); err != nil {
			l.Debug().Err(err).Msg("syncer saved, but failed to prepare new syncer")

			return err
		}
		l.Debug().Msg("syncer saved and new syncer will prepare")
	}

	return nil
}

func (sy *Syncers) isFinished() bool {
	switch {
	case sy.lastSyncer == nil:
		return true
	case sy.lastSyncer.State() != SyncerSaved:
		return false
	case sy.lastSyncer.HeightTo() == sy.targetHeight:
		return true
	default:
		return false
	}
}

func (sy *Syncers) mergeSourceNodes(ns []base.Node) {
	if len(sy.sourceNodes) < 1 {
		sy.sourceNodes = ns

		return
	}

	filtered := make([]base.Node, len(ns))
	for i := range ns {
		n := ns[i]

		var found bool
		for j := range sy.sourceNodes {
			s := sy.sourceNodes[j]
			if s.Address().Equal(n.Address()) {
				found = true

				break
			}
		}

		if !found {
			filtered[i] = n
		}
	}

	for i := range filtered {
		n := filtered[i]
		if n == nil {
			continue
		}

		sy.sourceNodes = append(sy.sourceNodes, n)
	}
}
