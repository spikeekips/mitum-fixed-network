package basicstates

import (
	"context"
	"sync"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type nodeInfoChecker struct {
	sync.RWMutex
	*logging.Logging
	*util.ContextDaemon
	policy        *isaac.LocalPolicy
	nodepool      *network.Nodepool
	interval      time.Duration
	lastHeight    base.Height
	whenNewHeight func(base.Height) error
}

func newNodeInfoChecker(
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	interval time.Duration,
	whenNewHeight func(base.Height) error,
) *nodeInfoChecker {
	nc := &nodeInfoChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "nodeinfo-checker")
		}),
		policy:        policy,
		nodepool:      nodepool,
		interval:      interval,
		whenNewHeight: whenNewHeight,
		lastHeight:    base.NilHeight,
	}
	nc.ContextDaemon = util.NewContextDaemon("nodeinfo-checker", nc.start)

	return nc
}

func (nc *nodeInfoChecker) SetLogger(logger logging.Logger) logging.Logger {
	_ = nc.ContextDaemon.SetLogger(logger)

	return nc.Logging.SetLogger(logger)
}

func (nc *nodeInfoChecker) start(ctx context.Context) error {
	if nc.interval < time.Second {
		n := config.DefaultSyncInterval

		nc.Log().Debug().Dur("interval", nc.interval).Dur("new_interval", n).Msg("interval too narrow; reset to default")

		nc.interval = n
	}

	ticker := time.NewTicker(nc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := nc.check(ctx); err != nil {
				return err
			}
		}
	}
}

func (nc *nodeInfoChecker) check(ctx context.Context) error {
	nctx, cancel := context.WithTimeout(ctx, nc.interval-time.Second)
	defer cancel()

	resultch := make(chan network.NodeInfo, nc.nodepool.LenRemotes())

	var wg sync.WaitGroup
	wg.Add(nc.nodepool.LenRemotes())
	nc.nodepool.TraverseRemotes(func(no network.Node) bool {
		go func(no network.Node) {
			defer wg.Done()

			resultch <- nc.request(nctx, no)
		}(no)

		return true
	})

	wg.Wait()
	close(resultch)

	for i := range resultch {
		if i == nil || i.LastBlock() == nil {
			continue
		}

		nc.newHeight(i.LastBlock().Height())
	}

	return nil
}

func (nc *nodeInfoChecker) newHeight(height base.Height) {
	nc.Lock()
	defer nc.Unlock()

	if nc.lastHeight >= height {
		return
	}

	nc.Log().Debug().Int64("height", height.Int64()).Msg("new height found")

	nc.lastHeight = height

	go func() {
		_ = nc.whenNewHeight(height)
	}()
}

func (nc *nodeInfoChecker) request(ctx context.Context, no network.Node) network.NodeInfo {
	l := nc.Log().WithLogger(func(lctx logging.Context) logging.Emitter {
		return lctx.Interface("node", no)
	})

	i, err := no.Channel().NodeInfo(ctx)
	if err != nil {
		l.Error().Err(err).Msg("failed to check nodeinfo")
	} else if err := nc.validateNodeInfo(no, i); err != nil {
		l.Error().Err(err).Msg("failed to validate nodeinfo")

		i = nil
	}

	return i
}

func (nc *nodeInfoChecker) validateNodeInfo(no network.Node, ni network.NodeInfo) error {
	if ni == nil {
		return xerrors.Errorf("empty nodeinfo")
	}

	if !no.Address().Equal(ni.Address()) {
		return xerrors.Errorf("address does not match: %q != %q", no.Address().String(), ni.Address().String())
	}

	if !nc.policy.NetworkID().Equal(ni.NetworkID()) {
		return xerrors.Errorf("network id does not match: %v != %v", nc.policy.NetworkID(), ni.NetworkID())
	}

	return nil
}

type SyncingStateNoneSuffrage struct {
	*BaseSyncingState
	nc *nodeInfoChecker
}

func NewSyncingStateNoneSuffrage(
	db storage.Database,
	blockData blockdata.BlockData,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	interval time.Duration,
) *SyncingStateNoneSuffrage {
	st := &SyncingStateNoneSuffrage{
		BaseSyncingState: NewBaseSyncingState("basic-syncing-state-none-suffrage", db, blockData, policy, nodepool),
	}

	st.nc = newNodeInfoChecker(policy, nodepool, interval, st.whenNewHeight)

	return st
}

func (st *SyncingStateNoneSuffrage) SetLogger(logger logging.Logger) logging.Logger {
	_ = st.nc.SetLogger(logger)

	return st.Logging.SetLogger(logger)
}

func (st *SyncingStateNoneSuffrage) Enter(sctx StateSwitchContext) (func() error, error) {
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

		return st.nc.Start()
	}, nil
}

func (st *SyncingStateNoneSuffrage) Exit(sctx StateSwitchContext) (func() error, error) {
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

		return st.nc.Stop()
	}, nil
}

func (st *SyncingStateNoneSuffrage) whenNewHeight(height base.Height) error {
	st.Lock()
	defer st.Unlock()

	if st.syncs == nil {
		return nil
	}

	sources := make([]network.Node, st.nodepool.LenRemotes())

	var i int
	st.nodepool.TraverseRemotes(func(no network.Node) bool {
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
