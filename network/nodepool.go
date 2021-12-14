package network

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/logging"
)

type passthroughItem struct {
	ch     Channel
	filter func(PassthroughedSeal) bool
}

// Nodepool contains all the known nodes including local node.
type Nodepool struct {
	*logging.Logging
	sync.RWMutex
	local   node.Local
	localch Channel
	nodes   map[string]base.Node
	chs     map[string]Channel
	pts     *cache.GCache // passthrough
}

func NewNodepool(local node.Local, ch Channel) *Nodepool {
	addr := local.Address().String()
	pts, _ := cache.NewGCache("lfu", 10, time.Minute)

	return &Nodepool{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "nodepool")
		}),
		local:   local,
		localch: ch,
		nodes: map[string]base.Node{
			addr: local,
		},
		chs: map[string]Channel{
			addr: ch,
		},
		pts: pts,
	}
}

func (np *Nodepool) Node(address base.Address) (base.Node, Channel, bool) {
	np.RLock()
	defer np.RUnlock()

	addr := address.String()
	n, found := np.nodes[addr]
	if !found {
		return nil, nil, false
	}

	return n, np.chs[addr], found
}

func (np *Nodepool) LocalNode() node.Local {
	return np.local
}

func (np *Nodepool) LocalChannel() Channel {
	return np.localch
}

func (np *Nodepool) Exists(address base.Address) bool {
	np.RLock()
	defer np.RUnlock()

	return np.exists(address)
}

func (np *Nodepool) Add(no base.Node, ch Channel) error {
	np.Lock()
	defer np.Unlock()

	addr := no.Address().String()
	if _, found := np.nodes[addr]; found {
		return util.FoundError.Errorf("already exists")
	}

	np.nodes[addr] = no
	np.chs[addr] = ch

	return nil
}

func (np *Nodepool) Channel(addr base.Address) (Channel, bool) {
	np.Lock()
	defer np.Unlock()

	ch, found := np.chs[addr.String()]

	return ch, found
}

func (np *Nodepool) SetChannel(addr base.Address, ch Channel) error {
	np.Lock()
	defer np.Unlock()

	if _, found := np.nodes[addr.String()]; !found {
		return util.NotFoundError.Errorf("unknown node, %q", addr)
	}

	np.chs[addr.String()] = ch

	if addr.Equal(np.local.Address()) {
		np.localch = ch
	}

	return nil
}

func (np *Nodepool) Remove(addrs ...base.Address) error {
	np.Lock()
	defer np.Unlock()

	founds := map[string]struct{}{}
	for _, addr := range addrs {
		if addr.Equal(np.local.Address()) {
			return errors.Errorf("local can not be removed, %q", addr)
		}

		if !np.exists(addr) {
			return errors.Errorf("Address does not exist, %q", addr)
		} else if _, found := founds[addr.String()]; found {
			return errors.Errorf("duplicated Address found, %q", addr)
		} else {
			founds[addr.String()] = struct{}{}
		}
	}

	for i := range addrs {
		addr := addrs[i].String()
		delete(np.nodes, addr)
		delete(np.chs, addr)
	}

	return nil
}

func (np *Nodepool) Len() int {
	np.RLock()
	defer np.RUnlock()

	return len(np.nodes)
}

func (np *Nodepool) LenRemoteAlives() int {
	var i int
	np.TraverseAliveRemotes(func(base.Node, Channel) bool {
		i++

		return true
	})

	return i
}

func (np *Nodepool) Traverse(callback func(base.Node, Channel) bool) {
	nodes, channels := np.nc(false)

	for i := range nodes {
		if !callback(nodes[i], channels[i]) {
			break
		}
	}
}

func (np *Nodepool) TraverseRemotes(callback func(base.Node, Channel) bool) {
	nodes, channels := np.nc(true)

	for i := range nodes {
		if !callback(nodes[i], channels[i]) {
			break
		}
	}
}

func (np *Nodepool) TraverseAliveRemotes(callback func(base.Node, Channel) bool) {
	nodes, channels := np.nc(true)

	for i := range nodes {
		ch := channels[i]
		if ch == nil {
			continue
		}

		if !callback(nodes[i], ch) {
			break
		}
	}
}

func (np *Nodepool) Broadcast( // revive:disable-line:function-length
	ctx context.Context,
	sl seal.Seal,
	filter func(base.Node) bool,
) ([]error, error) {
	l := np.Log().With().Stringer("seal_hash", sl.Hash()).Logger()

	var localci ConnInfo
	if ch := np.LocalChannel(); ch != nil {
		localci = ch.ConnInfo()
	}

	errch := make(chan error)
	wk := util.NewDistributeWorker(ctx, 100, errch)
	defer wk.Close()

	donech := make(chan []error, 1)
	go func() {
		var errs []error
		for i := range errch {
			if i != nil {
				errs = append(errs, i)
			}
		}

		donech <- errs
	}()

	targetch := make(chan [2]interface{})

	go func() {
		for i := range targetch {
			x, y := i[0], i[1]

			var no base.Node
			if x != nil {
				no = x.(base.Node)
			}
			ch := y.(Channel)

			if err := wk.NewJob(func(context.Context, uint64) error {
				return np.send(ctx, localci, no, ch, sl)
			}); err != nil {
				l.Trace().Err(err).Msg("something wrong to broadcast")

				break
			}
		}

		wk.Done()
	}()

	np.TraverseAliveRemotes(func(no base.Node, ch Channel) bool {
		if filter != nil && !filter(no) {
			return true
		}

		targetch <- [2]interface{}{no, ch}

		return true
	})

	np.passthroughs(func(ch Channel, filter func(PassthroughedSeal) bool) bool {
		if filter != nil && !filter(NewPassthroughedSealFromConnInfo(sl, localci)) {
			return true
		}

		targetch <- [2]interface{}{nil, ch}

		return true
	})

	close(targetch)

	if err := wk.Wait(); err != nil {
		close(errch)

		l.Trace().Err(err).Msg("failed to broadcast")

		return nil, err
	}

	close(errch)

	l.Trace().Msg("seal broadcasted")

	return <-donech, nil
}

func (np *Nodepool) ExistsPassthrough(ci ConnInfo) bool {
	return np.pts.Has(ci.String())
}

func (np *Nodepool) SetPassthrough(ch Channel, filter func(PassthroughedSeal) bool, expire time.Duration) error {
	np.Lock()
	defer np.Unlock()

	ci := ch.ConnInfo()
	if ci == nil {
		return fmt.Errorf("nil ConnInfo")
	}

	p, found := np.passthrough(ci)
	if found {
		// NOTE update filter
		p.ch = ch
		p.filter = filter

		return np.setPassthrough(p, expire)
	}

	for i := range np.chs {
		if np.chs[i] == nil {
			continue
		}

		eci := np.chs[i].ConnInfo()
		if eci == nil {
			continue
		}

		if eci.Equal(ci) {
			return util.FoundError.Errorf("already in ndoes")
		}
	}

	if filter == nil {
		filter = func(PassthroughedSeal) bool { return true }
	}

	return np.setPassthrough(passthroughItem{ch: ch, filter: filter}, expire)
}

func (np *Nodepool) RemovePassthrough(s string) error {
	np.Lock()
	defer np.Unlock()

	if !np.pts.Remove(s) {
		return util.NotFoundError.Call()
	}

	return nil
}

func (np *Nodepool) Passthroughs(ctx context.Context, sl PassthroughedSeal, callback func(seal.Seal, Channel)) error {
	wk := util.NewDistributeWorker(ctx, 100, nil)
	defer wk.Close()

	go func() {
		np.passthroughs(func(ch Channel, filter func(PassthroughedSeal) bool) bool {
			switch {
			case ch.ConnInfo().String() == sl.FromConnInfo():
				return true
			case filter == nil:
			case !filter(sl):
				return true
			}

			if err := wk.NewJob(func(context.Context, uint64) error {
				callback(sl.Seal, ch)

				return nil
			}); err != nil {
				return false
			}

			return true
		})

		wk.Done()
	}()

	return wk.Wait()
}

func (np *Nodepool) exists(address base.Address) bool {
	_, found := np.nodes[address.String()]

	return found
}

func (np *Nodepool) nc(filterLocal bool) ([]base.Node, []Channel) {
	np.RLock()
	defer np.RUnlock()

	if len(np.nodes) < 1 {
		return nil, nil
	}

	var d int
	if filterLocal {
		d = 1
	}

	nodes := make([]base.Node, len(np.nodes)-d)
	channels := make([]Channel, len(np.nodes)-d)

	addr := np.local.Address().String()
	var i int
	for k := range np.nodes {
		if filterLocal && k == addr {
			continue
		}

		nodes[i] = np.nodes[k]
		channels[i] = np.chs[k]

		i++
	}

	return nodes, channels
}

func (np *Nodepool) passthrough(ci ConnInfo) (passthroughItem, bool) {
	var p passthroughItem
	switch i, err := np.pts.Get(ci.String()); {
	case err != nil:
		return p, false
	case i == nil:
		return p, false
	default:
		return i.(passthroughItem), true
	}
}

func (np *Nodepool) setPassthrough(p passthroughItem, expire time.Duration) error {
	if expire <= 0 {
		return np.pts.SetWithoutExpire(p.ch.ConnInfo().String(), p)
	}

	return np.pts.Set(p.ch.ConnInfo().String(), p, expire)
}

type PassthroughedSeal struct {
	seal.Seal
	fromconnInfo string
}

func NewPassthroughedSealFromConnInfo(sl seal.Seal, ci ConnInfo) PassthroughedSeal {
	var s string
	if ci != nil {
		s = ci.String()
	}

	return NewPassthroughedSeal(sl, s)
}

func NewPassthroughedSeal(sl seal.Seal, ci string) PassthroughedSeal {
	return PassthroughedSeal{
		Seal:         sl,
		fromconnInfo: ci,
	}
}

func (sl PassthroughedSeal) FromConnInfo() string {
	return sl.fromconnInfo
}

func (np *Nodepool) passthroughs(callback func(Channel, func(PassthroughedSeal) bool) bool) {
	_ = np.pts.Traverse(func(_, v interface{}) bool {
		p, ok := v.(passthroughItem)
		if !ok {
			return true
		}

		return callback(p.ch, p.filter)
	})
}

func (np *Nodepool) send(ctx context.Context, localci ConnInfo, no base.Node, ch Channel, sl seal.Seal) error {
	e := np.Log().With().Stringer("seal_hash", sl.Hash())
	if no != nil {
		e = e.Stringer("target", no.Address())
	}
	e = e.Stringer("conninfo", ch.ConnInfo())

	l := e.Logger()

	switch err := ch.SendSeal(ctx, localci, sl); {
	case err == nil:
		l.Trace().Msg("seal broadcasted to node")

		return nil
	default:
		l.Trace().Err(err).Msg("failed to broadcasted to node")

		return fmt.Errorf("failed to broadcast seal to %q: %w", ch.ConnInfo(), err)
	}
}
