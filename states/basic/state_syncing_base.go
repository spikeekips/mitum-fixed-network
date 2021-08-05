package basicstates

import (
	"sort"
	"sync"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type BaseSyncingState struct {
	sync.RWMutex
	*logging.Logging
	*BaseState
	database  storage.Database
	blockData blockdata.BlockData
	policy    *isaac.LocalPolicy
	nodepool  *network.Nodepool
	syncs     *isaac.Syncers
}

func NewBaseSyncingState(
	name string,
	db storage.Database,
	blockData blockdata.BlockData,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
) *BaseSyncingState {
	return &BaseSyncingState{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", name)
		}),
		BaseState: NewBaseState(base.StateSyncing),
		database:  db,
		blockData: blockData,
		policy:    policy,
		nodepool:  nodepool,
	}
}

func (st *BaseSyncingState) Enter(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Enter(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	if st.syncers() != nil {
		return nil, xerrors.Errorf("not stopped correctly; syncers still running")
	}

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.enterCallback()
	}, nil
}

func (st *BaseSyncingState) Exit(sctx StateSwitchContext) (func() error, error) {
	callback := EmptySwitchFunc
	if i, err := st.BaseState.Exit(sctx); err != nil {
		return nil, err
	} else if i != nil {
		callback = i
	}

	syncs := st.syncers()
	st.setSyncers(nil)

	return func() error {
		if err := callback(); err != nil {
			return err
		}

		return st.exitCallback(syncs)
	}, nil
}

func (st *BaseSyncingState) enterCallback() error {
	var baseManifest block.Manifest
	if m, found, err := st.database.LastManifest(); err != nil {
		return err
	} else if found {
		baseManifest = m
	}

	syncs := isaac.NewSyncers(st.nodepool.LocalNode(), st.database, st.blockData, st.nodepool, st.policy, baseManifest)
	syncs.WhenBlockSaved(st.whenBlockSaved)
	syncs.WhenFinished(st.whenFinished)

	_ = syncs.SetLogging(st.Logging)

	if err := syncs.Start(); err != nil {
		return err
	}
	st.setSyncers(syncs)

	return nil
}

func (*BaseSyncingState) exitCallback(syncs *isaac.Syncers) error {
	if syncs == nil {
		return nil
	}

	return syncs.Stop()
}

func (st *BaseSyncingState) syncers() *isaac.Syncers {
	st.RLock()
	defer st.RUnlock()

	return st.syncs
}

func (st *BaseSyncingState) setSyncers(syncs *isaac.Syncers) {
	st.Lock()
	defer st.Unlock()

	st.syncs = syncs
}

func (st *BaseSyncingState) whenBlockSaved(blks []block.Block) {
	if len(blks) < 1 {
		panic("empty saved blocks in SyncingStateNoneSuffrage")
	}

	sort.Slice(blks, func(i, j int) bool {
		return blks[i].Height()-blks[j].Height() < 0
	})

	ivp := blks[len(blks)-1].ConsensusInfo().INITVoteproof()
	st.SetLastVoteproof(ivp)

	if err := st.NewBlocks(blks); err != nil {
		st.Log().Error().Err(err).Msg("new blocks hooks failed")
	}
}

func (st *BaseSyncingState) whenFinished(height base.Height) {
	st.Log().Debug().Int64("height", height.Int64()).Msg("syncing finished")
}
