package basicstates

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery"
	"github.com/spikeekips/mitum/states"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
)

type handoverState struct {
	sync.RWMutex
	uh bool
	on network.Channel
	ir bool
	gl sync.RWMutex
	g  bool
}

func (st *handoverState) underHandover() bool {
	st.RLock()
	defer st.RUnlock()

	return st.uh
}

func (st *handoverState) oldNode() network.Channel {
	st.RLock()
	defer st.RUnlock()

	return st.on
}

func (st *handoverState) isReady() bool {
	st.RLock()
	defer st.RUnlock()

	return st.ir
}

func (st *handoverState) setUnderHandover(b bool) *handoverState {
	if st.gLocked() {
		return st
	}

	st.Lock()
	defer st.Unlock()

	st.uh = b

	return st
}

func (st *handoverState) setOldNode(on network.Channel) {
	if st.gLocked() {
		return
	}

	st.Lock()
	defer st.Unlock()

	st.on = on
}

func (st *handoverState) setIsReady(b bool) *handoverState {
	if st.gLocked() {
		return st
	}

	st.Lock()
	defer st.Unlock()

	st.ir = b

	return st
}

func (st *handoverState) gLocked() bool {
	st.gl.RLock()
	defer st.gl.RUnlock()

	return st.g
}

func (st *handoverState) reset(f func() error) (func() /* unlock */, error) {
	st.gl.Lock()
	st.Lock()

	st.uh = false
	st.on = nil
	st.ir = false
	st.Unlock()

	st.g = true
	st.gl.Unlock()

	unlock := func() {
		st.gl.Lock()
		st.g = false
		st.gl.Unlock()
	}

	if f == nil {
		return unlock, nil
	}

	return unlock, f()
}

type DuplicatedError struct {
	ch network.Channel
	ni network.NodeInfo
}

func NewDuplicatedError(ch network.Channel, ni network.NodeInfo) DuplicatedError {
	return DuplicatedError{ch: ch, ni: ni}
}

func (DuplicatedError) Error() string {
	return "duplicated node found"
}

type handoverStoppedError struct {
	err error
	f   func()
}

func (err handoverStoppedError) Error() string {
	if err.err == nil {
		return ""
	}

	return err.err.Error()
}

type Handover struct {
	sync.Mutex
	*logging.Logging
	*util.ContextDaemon
	ci                                     network.ConnInfo
	encs                                   *encoder.Encoders
	policy                                 *isaac.LocalPolicy
	nodepool                               *network.Nodepool
	suffrage                               base.Suffrage
	checkDuplicatedNodeFunc                func() (network.Channel, network.NodeInfo, error)
	st                                     *handoverState
	rchs                                   map[string]network.Channel
	intervalKeepVerifyDuplicatedNode       time.Duration
	maxFailedCountKeepVerifyDuplicatedNode uint
	intervalPingHandover                   time.Duration
}

func NewHandoverWithDiscoveryURL(
	localci network.ConnInfo,
	encs *encoder.Encoders,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
	cis []network.ConnInfo,
) (*Handover, error) {
	rchs := map[string]network.Channel{}
	for i := range cis {
		ci := cis[i]

		ch, err := discovery.LoadNodeChannel(ci, encs, policy.NetworkConnectionTimeout())
		if err != nil {
			return nil, err
		}

		rchs[ci.String()] = ch
	}

	hd := NewHandover(localci, encs, policy, nodepool, suffrage)
	hd.rchs = rchs

	return hd, nil
}

func NewHandover(
	localci network.ConnInfo,
	encs *encoder.Encoders,
	policy *isaac.LocalPolicy,
	nodepool *network.Nodepool,
	suffrage base.Suffrage,
) *Handover {
	hd := &Handover{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "basic-handover")
		}),
		ci:                                     localci,
		encs:                                   encs,
		policy:                                 policy,
		nodepool:                               nodepool,
		suffrage:                               suffrage,
		st:                                     &handoverState{},
		rchs:                                   map[string]network.Channel{},
		intervalKeepVerifyDuplicatedNode:       time.Second * 2,
		maxFailedCountKeepVerifyDuplicatedNode: 3,
		intervalPingHandover:                   states.DefaultPingHandoverInterval,
	}

	hd.ContextDaemon = util.NewContextDaemon("handover", func(ctx context.Context) error {
		return hd.start(ctx, nil)
	})

	return hd
}

func (hd *Handover) Start() error {
	hd.Lock()
	defer hd.Unlock()

	return hd.startUntilUnderHandover()
}

func (hd *Handover) Refresh(chs ...network.Channel) error {
	hd.Lock()
	defer hd.Unlock()

	hd.Log().Debug().Msg("trying to refresh")

	switch added, err := hd.addRemoteChannels(chs); {
	case err != nil:
		return fmt.Errorf("failed to refresh: %w", err)
	case len(chs) > 0:
		hd.Log().Debug().Bool("added", added).Msg("remote channel added")
	}

	f, err := hd.stop()
	f()

	if err != nil {
		hd.Log().Error().Err(err).Msg("failed to refresh")

		return err
	}

	if err := hd.startUntilUnderHandover(); err != nil {
		hd.Log().Error().Err(err).Msg("failed to refresh")

		return err
	}

	hd.Log().Debug().Msg("refreshed")

	return nil
}

func (hd *Handover) Stop() error {
	hd.Lock()
	defer hd.Unlock()

	var f func()
	unlock, err := hd.st.reset(func() error {
		i, err := hd.stop()
		f = i

		return err
	})

	go func() {
		f()
		unlock()
	}()

	return err
}

func (hd *Handover) UnderHandover() bool {
	return hd.st.underHandover()
}

func (hd *Handover) SetLogging(l *logging.Logging) *logging.Logging {
	_ = hd.ContextDaemon.SetLogging(l)

	return hd.Logging.SetLogging(l)
}

func (hd *Handover) startUntilUnderHandover() error {
	investigatedch := make(chan bool, 1)
	hd.ContextDaemon = util.NewContextDaemon("handover", func(ctx context.Context) error {
		return hd.start(ctx, investigatedch)
	})
	_ = hd.ContextDaemon.SetLogging(hd.Logging)

	if err := hd.ContextDaemon.Start(); err != nil {
		return err
	}

	<-investigatedch

	return nil
}

func (hd *Handover) start(ctx context.Context, investigatedch chan bool) error {
	if hd.checkDuplicatedNodeFunc == nil {
		hd.checkDuplicatedNodeFunc = hd.defaultCheckDuplicatedNode
	}

	underhandover, err := hd.investigate()
	if investigatedch != nil {
		investigatedch <- underhandover
	}

	if err != nil {
		unlock, _ := hd.st.reset(nil)
		defer unlock()

		hd.Log().Error().Err(err).Msg("failed to investigate handover")

		return err
	}

	hd.Log().Debug().Bool("underhandover", underhandover).Msg("investigated")

	if !underhandover {
		unlock, _ := hd.st.reset(nil)
		defer unlock()

		return nil
	}

	nctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if err := hd.startPing(nctx, cancel); err != nil {
		return fmt.Errorf("failed to start ping: %w", err)
	}

	donech := make(chan struct{}, 1)
	go hd.keepVerifyDuplicatedNode(nctx, cancel, donech)

	<-nctx.Done()

	return func() error {
		return handoverStoppedError{err: nctx.Err(), f: func() {
			<-donech
		}}
	}()
}

func (hd *Handover) stop() (func(), error) {
	emptyfunc := func() {}

	err := hd.ContextDaemon.Stop()

	var he handoverStoppedError
	switch {
	case err == nil:
		return emptyfunc, nil
	case errors.As(err, &he):
	case errors.Is(err, util.DaemonAlreadyStoppedError):
		return emptyfunc, nil
	default:
		return emptyfunc, err
	}

	if errors.Is(he.err, util.DaemonAlreadyStoppedError) {
		he.err = nil
	}

	return he.f, he.err
}

// IsReady indicates node operator approves Handover.
func (hd *Handover) IsReady() bool {
	return hd.UnderHandover() && hd.st.isReady()
}

func (hd *Handover) setReady(b bool) {
	_ = hd.st.setIsReady(b)

	hd.Log().Debug().Msg("handover started")
}

func (hd *Handover) OldNode() network.Channel {
	return hd.st.oldNode()
}

func (hd *Handover) setOldNode(ch network.Channel) {
	switch i := hd.st.oldNode(); {
	case i == nil:
		if ch == nil {
			return
		}
	case ch == nil:
		hd.st.setOldNode(nil)

		return
	case ch.ConnInfo().Equal(i.ConnInfo()):
		return
	}

	l, ok := ch.(logging.SetLogging)
	if ok {
		_ = l.SetLogging(hd.Logging)
	}

	hd.st.setOldNode(ch)
}

func (hd *Handover) investigate() (bool, error) {
	if !hd.suffrage.IsInside(hd.nodepool.LocalNode().Address()) {
		hd.Log().Debug().Msg("local is not suffrage node; no need to find duplicated node")

		return false, util.IgnoreError.Errorf("local is not suffrage node")
	}

	if len(hd.remoteChannels()) < 1 {
		hd.Log().Debug().Msg("empty remote channsls")

		return false, nil
	}

	max := 3
	var tried int

	var ni network.NodeInfo
	var ch network.Channel

end:
	for {
		var err error

		ch, ni, err = hd.checkDuplicatedNodeFunc()
		switch {
		case errors.Is(err, util.IgnoreError):
			break end
		case err != nil:
			tried++

			if tried >= max {
				return false, err
			}
		default:
			break end
		}
	}

	if ch == nil {
		hd.Log().Debug().Msg("duplicated node not found")

		return false, nil
	}

	if err := hd.whenFound(ch, ni); err != nil {
		return false, err
	}

	return true, nil
}

func (hd *Handover) defaultCheckDuplicatedNode() (network.Channel, network.NodeInfo, error) {
	chs := hd.remoteChannels()
	if len(chs) < 1 {
		return nil, nil, nil
	}

	hd.Log().Debug().Func(func(e *zerolog.Event) {
		for i := range chs {
			_ = e.Stringer("ch#"+i, chs[i].ConnInfo())
		}
	}).Msg("trying to check duplicated node")

	wk := util.NewErrgroupWorker(context.Background(), 100)
	defer wk.Close()

	go func() {
		defer wk.Done()

		for i := range chs {
			ch := chs[i]

			if err := wk.NewJob(func(ctx context.Context, _ uint64) error {
				l := hd.Log().With().Stringer("conninfo", ch.ConnInfo()).Logger()

				ni, err := ch.NodeInfo(ctx)
				if err != nil {
					l.Error().Err(err).Msg("failed to get node info")

					return nil
				}
				if err := ni.IsValid(nil); err != nil {
					l.Error().Err(err).Msg("failed to get node info; invalid NodeInfo")

					return nil
				}

				dup, found := hd.findDuplicatedNodeFromNodeInfo(ctx, ni)
				if !found {
					return nil
				}

				l.Error().Msg("duplicated node found; same node is already running")

				return NewDuplicatedError(dup, ni)
			}); err != nil {
				hd.Log().Error().Err(err).Msg("failed to new job")

				return
			}
		}
	}()

	err := wk.Wait()

	var derr DuplicatedError
	switch {
	case err == nil:
		return nil, nil, util.IgnoreError.Errorf("failed to find duplicated node")
	case !errors.As(err, &derr):
		return nil, nil, err
	}

	return derr.ch, derr.ni, nil
}

func (hd *Handover) findDuplicatedNodeFromNodeInfo(ctx context.Context, ni network.NodeInfo) (network.Channel, bool) {
	if ni.Address().Equal(hd.nodepool.LocalNode().Address()) {
		ch, err := discovery.LoadNodeChannel(ni.ConnInfo(), hd.encs, hd.policy.NetworkConnectionTimeout())
		if err != nil {
			return nil, false
		}

		return ch, true
	}

	nodes := ni.Nodes()
	if len(nodes) < 1 {
		return nil, false
	}

	var dup network.ConnInfo
	for i := range nodes {
		no := nodes[i]
		if no.Address.Equal(hd.nodepool.LocalNode().Address()) {
			ci := no.ConnInfo()
			if ci != nil {
				dup = ci
			}

			break
		}
	}

	if dup == nil {
		return nil, false
	}

	ch, err := discovery.LoadNodeChannel(dup, hd.encs, hd.policy.NetworkConnectionTimeout())
	if err != nil {
		return nil, false
	}

	hd.Log().Error().Interface("conninfo", ch.ConnInfo()).Msg("duplication suspected node found")

	if _, err := ch.NodeInfo(ctx); err != nil {
		return nil, false
	}

	return ch, true
}

func (hd *Handover) keepVerifyDuplicatedNode(ctx context.Context, cancel context.CancelFunc, donech chan struct{}) {
	defer func() {
		donech <- struct{}{}
	}()

	on := hd.OldNode()
	if on == nil {
		return
	}

	ticker := time.NewTicker(hd.intervalKeepVerifyDuplicatedNode)
	defer ticker.Stop()

	max := hd.maxFailedCountKeepVerifyDuplicatedNode
	if max < 1 {
		max = 1
	}

	var failed uint

end:
	for {
		select {
		case <-ctx.Done():
			break end
		case <-ticker.C:
			switch ni, err := on.NodeInfo(ctx); {
			case ni == nil:
			case err == nil && ni.Address().Equal(hd.nodepool.LocalNode().Address()):
				if err := hd.whenFound(on, ni); err != nil {
					continue
				}

				failed = 0

				continue
			}

			failed++
			if failed >= max {
				unlock, _ := hd.st.reset(nil)
				unlock()

				hd.Log().Debug().Msg("old node is not alive; not under handover")

				cancel()

				failed = 0
			}
		}
	}
}

func (hd *Handover) startPing(ctx context.Context, _ context.CancelFunc) error {
	hd.Log().Debug().Msg("starting ping")

	old := hd.OldNode()
	if old == nil {
		return errors.Errorf("old node is not set")
	}

	if _, err := hd.pingSeal(); err != nil {
		return fmt.Errorf("failed to make PingHandoverSeal: %w", err)
	}

	timer := localtime.NewContextTimer("ping", hd.intervalPingHandover, func(int) (bool, error) {
		if !hd.UnderHandover() {
			hd.Log().Debug().Stringer("conninfo", old.ConnInfo()).Msg("old node is not alive; ping passed")

			return true, nil
		}

		old := hd.OldNode()
		if old == nil {
			return false, errors.Errorf("old node is not set")
		}

		sl, err := hd.pingSeal()
		if err != nil {
			return false, fmt.Errorf("failed to make PingHandoverSeal: %w", err)
		}

		switch ok, err := old.PingHandover(context.Background(), sl); {
		case err != nil:
			hd.Log().Error().Err(err).Stringer("conninfo", old.ConnInfo()).Msg("failed to ping")
		case !ok:
			hd.Log().Error().Stringer("conninfo", old.ConnInfo()).Msg("failed to ping; old node rejects")
		}

		return true, nil
	})

	return timer.StartWithContext(ctx)
}

func (hd *Handover) pingSeal() (network.HandoverSeal, error) {
	return network.NewHandoverSealV0(
		network.PingHandoverSealV0Hint,
		hd.nodepool.LocalNode().Privatekey(),
		hd.nodepool.LocalNode().Address(),
		hd.ci,
		hd.policy.NetworkID(),
	)
}

func (hd *Handover) endHandoverSeal() (network.HandoverSeal, error) {
	return network.NewHandoverSealV0(
		network.EndHandoverSealV0Hint,
		hd.nodepool.LocalNode().Privatekey(),
		hd.nodepool.LocalNode().Address(),
		hd.ci,
		hd.policy.NetworkID(),
	)
}

func (hd *Handover) updateNodes(nodes []network.RemoteNode) error {
	var updated []network.RemoteNode
	for i := range nodes {
		no := nodes[i]
		switch {
		case !hd.suffrage.IsInside(no.Address):
			continue
		case no.ConnInfo() == nil:
			continue
		case no.Address.Equal(hd.nodepool.LocalNode().Address()):
			continue
		}

		ch, err := discovery.LoadNodeChannel(no.ConnInfo(), hd.encs, hd.policy.NetworkConnectionTimeout())
		if err != nil {
			return fmt.Errorf("failed to load channel from nodeinfo: %w", err)
		}

		if !hd.nodepool.Exists(no.Address) {
			updated = append(updated, no)
			if err := hd.nodepool.Add(node.NewRemote(no.Address, no.Publickey), ch); err != nil {
				return err
			}

			continue
		}

		if ech, _ := hd.nodepool.Channel(no.Address); ech == nil || !ech.ConnInfo().Equal(no.ConnInfo()) {
			updated = append(updated, no)

			if err := hd.nodepool.SetChannel(no.Address, ch); err != nil {
				return err
			}
		}
	}

	if len(updated) > 0 {
		hd.Log().Debug().Interface("updated", updated).Msg("nodes in nodepool updated")
	}

	return nil
}

func (hd *Handover) remoteChannels() map[string]network.Channel {
	founds := map[string]network.Channel{}
	hd.nodepool.TraverseAliveRemotes(func(_ base.Node, ch network.Channel) bool {
		ci := ch.ConnInfo().String()
		if _, found := founds[ci]; found {
			return true
		}

		founds[ci] = ch

		return true
	})

	for i := range hd.rchs {
		ch := hd.rchs[i]
		ci := ch.ConnInfo().String()
		if _, found := founds[ci]; found {
			continue
		}

		founds[ci] = ch
	}

	return founds
}

func (hd *Handover) addRemoteChannels(chs []network.Channel) (bool, error) {
	if len(chs) < 1 {
		return false, nil
	}

	rchs := hd.remoteChannels()

	var added bool
	for i := range chs {
		ch := chs[i]
		if ch == nil || ch.ConnInfo() == nil {
			return false, errors.Errorf("failed to add remote channel; empty channel found")
		}

		if _, found := rchs[ch.ConnInfo().String()]; found {
			continue
		}

		hd.rchs[ch.ConnInfo().String()] = ch
		added = true
	}

	return added, nil
}

func (hd *Handover) loadChannel(ci network.ConnInfo) (network.Channel, error) {
	return discovery.LoadNodeChannel(ci, hd.encs, hd.policy.NetworkConnectionTimeout())
}

func (hd *Handover) whenFound(ch network.Channel, ni network.NodeInfo) error {
	if ni != nil {
		if err := hd.updateNodes(ni.Nodes()); err != nil {
			hd.Log().Error().Err(err).Msg("failed to update nodes in nodepool")

			return err
		}
	}

	_ = hd.st.setUnderHandover(true)
	hd.setOldNode(ch)

	return nil
}
