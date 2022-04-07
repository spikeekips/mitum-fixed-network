package network

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type NodeInfoChecker struct {
	sync.RWMutex
	*logging.Logging
	*util.ContextDaemon
	networkID     base.NetworkID
	nodepool      *Nodepool
	interval      time.Duration
	lastHeight    base.Height
	whenNewHeight func(base.Height) error
}

func NewNodeInfoChecker(
	networkID base.NetworkID,
	nodepool *Nodepool,
	interval time.Duration,
	whenNewHeight func(base.Height) error,
) *NodeInfoChecker {
	if whenNewHeight == nil {
		whenNewHeight = func(base.Height) error { return nil }
	}

	nc := &NodeInfoChecker{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "nodeinfo-checker")
		}),
		networkID:     networkID,
		nodepool:      nodepool,
		interval:      interval,
		whenNewHeight: whenNewHeight,
		lastHeight:    base.NilHeight,
	}
	nc.ContextDaemon = util.NewContextDaemon("nodeinfo-checker", nc.start)

	return nc
}

func (nc *NodeInfoChecker) SetLogging(l *logging.Logging) *logging.Logging {
	_ = nc.ContextDaemon.SetLogging(l)

	return nc.Logging.SetLogging(l)
}

func (nc *NodeInfoChecker) start(ctx context.Context) error {
	if nc.interval < time.Second {
		n := time.Second * 2

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

func (nc *NodeInfoChecker) check(ctx context.Context) error {
	if nc.nodepool.LenRemoteAlives() < 1 {
		return nil
	}

	interval := nc.interval - time.Second
	if interval < time.Second*2 {
		interval = time.Second * 2
	}
	nctx, cancel := context.WithTimeout(ctx, interval)
	defer cancel()

	lenremotes := nc.nodepool.Len() - 1
	resultch := make(chan NodeInfo, lenremotes)

	var wg sync.WaitGroup
	wg.Add(lenremotes)
	nc.nodepool.TraverseAliveRemotes(func(no base.Node, ch Channel) bool {
		go func(no base.Node, ch Channel) {
			defer wg.Done()

			resultch <- nc.request(nctx, no, ch)
		}(no, ch)

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

func (nc *NodeInfoChecker) newHeight(height base.Height) {
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

func (nc *NodeInfoChecker) request(ctx context.Context, no base.Node, ch Channel) NodeInfo {
	l := nc.Log().With().Interface("node", no).Logger()

	i, err := ch.NodeInfo(ctx)
	if err != nil {
		l.Error().Err(err).Msg("failed to check nodeinfo")
	} else if err := nc.validateNodeInfo(no, i); err != nil {
		l.Error().Err(err).Msg("failed to validate nodeinfo")

		i = nil
	}

	return i
}

func (nc *NodeInfoChecker) validateNodeInfo(no base.Node, ni NodeInfo) error {
	if ni == nil {
		return errors.Errorf("empty nodeinfo")
	}

	if err := ni.IsValid(nil); err != nil {
		return err
	}

	if !no.Address().Equal(ni.Address()) {
		return errors.Errorf("address does not match: %q != %q", no.Address().String(), ni.Address().String())
	}

	if !nc.networkID.Equal(ni.NetworkID()) {
		return errors.Errorf("network id does not match: %v != %v", nc.networkID, ni.NetworkID())
	}

	return nil
}
