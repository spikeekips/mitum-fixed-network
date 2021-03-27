package isaac

import (
	"context"
	"sync"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type Syncers struct {
	sync.RWMutex
	*util.ContextDaemon
	*logging.Logging
	local                *network.LocalNode
	database             storage.Database
	blockData            blockdata.BlockData
	policy               *LocalPolicy
	baseManifest         block.Manifest
	limitBlocksPerSyncer uint
	stateChan            chan SyncerStateChangedContext
	whenFinished         func(base.Height)
	whenBlockSaved       func([]block.Block)
	targetHeight         base.Height
	prevSyncer           Syncer
	lastSyncer           Syncer
	sourceNodes          []network.Node
}

func NewSyncers(
	local *network.LocalNode,
	st storage.Database,
	blockData blockdata.BlockData,
	policy *LocalPolicy,
	baseManifest block.Manifest,
) *Syncers {
	sy := &Syncers{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "syncers")
		}),
		local:                local,
		database:             st,
		blockData:            blockData,
		policy:               policy,
		baseManifest:         baseManifest,
		limitBlocksPerSyncer: 10,
		stateChan:            make(chan SyncerStateChangedContext),
		whenFinished:         func(base.Height) {},
		whenBlockSaved:       func([]block.Block) {},
		targetHeight:         base.NilHeight,
	}

	sy.ContextDaemon = util.NewContextDaemon("syncers", sy.start)

	return sy
}

func (sy *Syncers) Stop() error {
	sy.Lock()
	defer sy.Unlock()

	if err := sy.ContextDaemon.Stop(); err != nil {
		if !xerrors.Is(err, util.DaemonAlreadyStoppedError) {
			return err
		}
	}

	if sy.lastSyncer != nil {
		if err := sy.lastSyncer.Close(); err != nil {
			return xerrors.Errorf("failed to close last syncer: %w", err)
		}

		sy.lastSyncer = nil
	}

	if sy.prevSyncer != nil {
		if err := sy.prevSyncer.Close(); err != nil {
			return xerrors.Errorf("failed to close last syncer: %w", err)
		}

		sy.prevSyncer = nil
	}

	return nil
}

func (sy *Syncers) SetLogger(l logging.Logger) logging.Logger {
	_ = sy.Logging.SetLogger(l)
	_ = sy.ContextDaemon.SetLogger(l)

	return sy.Log()
}

func (sy *Syncers) WhenFinished(callback func(base.Height)) {
	sy.whenFinished = callback
}

func (sy *Syncers) WhenBlockSaved(callback func([]block.Block)) {
	sy.whenBlockSaved = callback
}

// Add adds new syncer with target height. If it returns true, it means Syncers
// not yet finished.
func (sy *Syncers) Add(to base.Height, sourceNodes []network.Node) (bool, error) {
	sy.Lock()
	defer sy.Unlock()

	isFinished := sy.isFinished()

	if to <= sy.targetHeight {
		return isFinished, util.IgnoreError.Errorf("lower height")
	}

	l := sy.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Hinted("previous_height", sy.targetHeight).Hinted("new_height", to)
	})

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

	if i, err := sy.newSyncer(baseManifest); err != nil {
		l.Debug().Msg("target height updated, but failed to add new syncer")

		return false, err
	} else {
		if sy.lastSyncer != nil {
			sy.prevSyncer = sy.lastSyncer
		}

		sy.lastSyncer = i

		l.Debug().Msg("target height updated and new syncer added")

		go func() {
			sy.stateChan <- NewSyncerStateChangedContext(i, SyncerCreated, nil)
		}()
	}

	return false, nil
}

func (sy *Syncers) IsFinished() bool {
	sy.RLock()
	defer sy.RUnlock()

	return sy.isFinished()
}

func (sy *Syncers) start(ctx context.Context) error {
end:
	for {
		select {
		case <-ctx.Done():
			break end
		case cxt := <-sy.stateChan:
			if err := sy.stateChanged(cxt); err != nil {
				sy.Log().Error().Err(err).Msg("failed to handle state changed")
			}
		}
	}

	return nil
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

	l := sy.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		var from base.Height
		if baseManifest == nil {
			from = base.PreGenesisHeight
		} else {
			from = baseManifest.Height() + 1
		}

		return ctx.Hinted("from", from).Hinted("to", to)
	})

	var syncer Syncer
	if i, err := NewGeneralSyncer(
		sy.local,
		sy.database,
		sy.blockData,
		sy.policy,
		sy.sourceNodes,
		baseManifest, to,
	); err != nil {
		l.Debug().Msg("failed to add new syncer")

		return nil, err
	} else {
		i = i.SetStateChan(sy.stateChan)

		syncer = i
	}

	if l, ok := syncer.(logging.SetLogger); ok {
		_ = l.SetLogger(sy.Log())
	}

	l.Debug().Msg("new syncer added")

	return syncer, nil
}

func (sy *Syncers) prepareSyncer(baseManifest block.Manifest) error {
	l := sy.Log().WithLogger(func(lctx logging.Context) logging.Emitter {
		var from base.Height
		if baseManifest == nil {
			from = base.PreGenesisHeight
		} else {
			from = baseManifest.Height() + 1
		}

		return lctx.Hinted("from", from)
	})

	var newSyncer Syncer
	if i, err := sy.newSyncer(baseManifest); err != nil {
		if xerrors.Is(err, util.IgnoreError) {
			l.Debug().Err(err).Msg("failed to make new syncer")

			return nil
		} else {
			l.Error().Err(err).Msg("failed to make new syncer")

			return err
		}
	} else {
		newSyncer = i
	}

	if err := newSyncer.Prepare(); err != nil {
		l.Error().Err(err).Msg("failed to prepare syncer")

		return err
	} else {
		if sy.lastSyncer != nil {
			sy.prevSyncer = sy.lastSyncer
		}

		sy.lastSyncer = newSyncer

		l.Debug().Msg("new syncer will prepare")

		return nil
	}
}

func (sy *Syncers) stateChanged(ctx SyncerStateChangedContext) error {
	sy.Lock()
	defer sy.Unlock()

	syncer := ctx.Syncer()
	l := sy.Log().WithLogger(func(lctx logging.Context) logging.Emitter {
		return lctx.
			Str("syncer", syncer.ID()).
			Str("state", ctx.State().String()).
			Hinted("from", syncer.HeightFrom()).
			Hinted("to", syncer.HeightTo())
	})

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
	default:
		l.Debug().Msg("syncer state changed")
	}

	return nil
}

func (sy *Syncers) stateChangedCreated(ctx SyncerStateChangedContext) error {
	syncer := ctx.Syncer()

	l := sy.Log().WithLogger(func(lctx logging.Context) logging.Emitter {
		return lctx.
			Str("syncer", syncer.ID()).
			Str("state", ctx.State().String()).
			Hinted("from", syncer.HeightFrom()).
			Hinted("to", syncer.HeightTo())
	})

	if err := syncer.Prepare(); err != nil {
		l.Error().Err(err).Msg("failed to prepare")

		return err
	} else {
		l.Debug().Msg("new syncer will prepare")
	}

	return nil
}

func (sy *Syncers) stateChangedPrepared(ctx SyncerStateChangedContext) error {
	syncer := ctx.Syncer()

	l := sy.Log().WithLogger(func(lctx logging.Context) logging.Emitter {
		return lctx.
			Str("syncer", syncer.ID()).
			Str("state", ctx.State().String()).
			Hinted("from", syncer.HeightFrom()).
			Hinted("to", syncer.HeightTo())
	})

	l.Debug().Msg("syncer prepared")

	return sy.prepareSyncer(syncer.TailManifest())
}

func (sy *Syncers) stateChangedSaved(ctx SyncerStateChangedContext) error {
	syncer := ctx.Syncer()

	l := sy.Log().WithLogger(func(lctx logging.Context) logging.Emitter {
		return lctx.
			Str("syncer", syncer.ID()).
			Str("state", ctx.State().String()).
			Hinted("from", syncer.HeightFrom()).
			Hinted("to", syncer.HeightTo()).
			Hinted("target_height", sy.targetHeight).
			Hinted("last_syncer_height", sy.lastSyncer.HeightTo())
	})

	if st, ok := sy.database.(storage.LastBlockSaver); ok {
		if err := st.SaveLastBlock(syncer.HeightTo()); err != nil {
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
		} else {
			l.Debug().Msg("syncer saved and new syncer will prepare")
		}
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

func (sy *Syncers) mergeSourceNodes(ns []network.Node) {
	if len(sy.sourceNodes) < 1 {
		sy.sourceNodes = ns

		return
	}

	filtered := make([]network.Node, len(ns))
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
