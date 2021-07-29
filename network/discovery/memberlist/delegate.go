package memberlist

import (
	"sync"

	ml "github.com/hashicorp/memberlist"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type Events struct {
	sync.RWMutex
	*logging.Logging
	connInfo              ConnInfo
	meta                  NodeMeta
	ma                    *ConnInfoMap
	notifyJoin            func(*ml.Node, NodeMeta)
	notifyLeave           func(*ml.Node, NodeMeta)
	notifyUpdate          func(*ml.Node, NodeMeta)
	notifyMerge           func([]*ml.Node, map[string]NodeMeta) error
	discoveryNotifyJoin   func(*ml.Node, NodeMeta) error
	discoveryNotifyLeave  func(*ml.Node, NodeMeta) error
	discoveryNotifyUpdate func(*ml.Node, NodeMeta)
	discoveryNotifyMerge  func([]*ml.Node, map[string]NodeMeta) error
	notifyAlive           func(*ml.Node, NodeMeta) error
	requestNodeMeta       func(NodeMeta) []byte
	notifyMsg             func([]byte)
	localState            func(join bool) []byte
	mergeRemoteState      func(buf []byte, join bool)
}

func NewEvents(connInfo ConnInfo, meta NodeMeta, ma *ConnInfoMap) *Events {
	return &Events{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "memberlist-discovery-event-delegate")
		}),
		connInfo:              connInfo,
		meta:                  meta,
		ma:                    ma,
		notifyJoin:            func(*ml.Node, NodeMeta) {},
		notifyLeave:           func(*ml.Node, NodeMeta) {},
		notifyUpdate:          func(*ml.Node, NodeMeta) {},
		notifyMerge:           func([]*ml.Node, map[string]NodeMeta) error { return nil },
		discoveryNotifyJoin:   func(*ml.Node, NodeMeta) error { return nil },
		discoveryNotifyLeave:  func(*ml.Node, NodeMeta) error { return nil },
		discoveryNotifyUpdate: func(*ml.Node, NodeMeta) {},
		discoveryNotifyMerge:  func([]*ml.Node, map[string]NodeMeta) error { return nil },
		notifyAlive:           func(*ml.Node, NodeMeta) error { return nil },
		requestNodeMeta:       func(meta NodeMeta) []byte { return meta.Bytes() },
		notifyMsg:             func([]byte) {},
		localState:            func(join bool) []byte { return nil },
		mergeRemoteState:      func(buf []byte, join bool) {},
	}
}

func (dg *Events) NotifyJoin(peer *ml.Node) {
	meta, l, err := dg.loadMeta("notify_join", peer)
	if err != nil {
		return
	}

	addr := peer.Address()

	l.Debug().Bool("known", dg.ma.addrExists(addr)).Msg("notify join")

	ci, _ := dg.ma.dryAdd(meta.Publish(), meta.Insecure())
	if addr != ci.Address {
		l.Error().Str("conn_address", ci.Address).Msg("wrong address of new node")

		return
	}

	_, _ = dg.ma.add(meta.Publish(), meta.Insecure())

	if err := dg.discoveryNotifyJoin(peer, meta); err != nil {
		dg.Log().Error().Err(err).Msg("failed to join node")

		return
	}

	if dg.notifyJoin != nil {
		dg.notifyJoin(peer, meta)
	}
}

func (dg *Events) SetNotifyJoin(callback func(peer *ml.Node, meta NodeMeta)) *Events {
	dg.notifyJoin = callback

	return dg
}

func (dg *Events) NotifyLeave(peer *ml.Node) {
	meta, l, err := dg.loadMeta("notify_leave", peer)
	if err != nil {
		return
	}

	removed := dg.ma.remove(peer.Address())

	l.Debug().Bool("removed_from_connmap", removed).Msg("notify leave")

	if err := dg.discoveryNotifyLeave(peer, meta); err != nil {
		dg.Log().Error().Err(err).Msg("failed to leave node")

		return
	}

	dg.notifyLeave(peer, meta)
}

func (dg *Events) SetNotifyLeave(callback func(peer *ml.Node, meta NodeMeta)) *Events {
	dg.notifyLeave = callback

	return dg
}

func (dg *Events) NotifyUpdate(peer *ml.Node) {
	meta, l, err := dg.loadMeta("notify_update", peer)
	if err != nil {
		return
	}

	l.Debug().Msg("notify update")

	dg.discoveryNotifyUpdate(peer, meta)

	dg.notifyUpdate(peer, meta)
}

func (dg *Events) SetNotifyUpdate(callback func(peer *ml.Node, meta NodeMeta)) *Events {
	dg.notifyUpdate = callback

	return dg
}

func (dg *Events) NotifyMerge(peers []*ml.Node) error {
	metas := map[string]NodeMeta{}

	le := dg.Log().Debug()
	for i := range peers {
		p := peers[i]
		addr := p.Address()

		meta, l, err := dg.loadMeta("notify_merge", p)
		if err != nil {
			l.Error().Err(err).Msg("failed to parse meta")

			return err
		}

		le.Dict("peer_"+addr, LogNodeMeta(LogNode(logging.Dict(), p), meta))
		metas[addr] = meta
	}

	le.Msg("notify merge")

	if err := dg.discoveryNotifyMerge(peers, metas); err != nil {
		dg.Log().Error().Err(err).Msg("failed to merge nodes")

		return err
	}

	return dg.notifyMerge(peers, metas)
}

func (dg *Events) SetNotifyMerge(callback func(peers []*ml.Node, metas map[string]NodeMeta) error) *Events {
	dg.notifyMerge = callback

	return dg
}

// NotifyAlive does not do anything to filter node; if error occurs, alive
// mesage will be ignored.
func (dg *Events) NotifyAlive(peer *ml.Node) error {
	addr := peer.Address()
	if addr == dg.connInfo.Address {
		return nil
	}

	meta, l, err := dg.loadMeta("notify_alive", peer)
	if err != nil {
		return err
	}

	l.Debug().Msg("notify alive")

	if !dg.ma.addrExists(addr) {
		_, _ = dg.ma.add(meta.Publish(), meta.Insecure())
	}

	return dg.notifyAlive(peer, meta)
}

func (dg *Events) SetNotifyAlive(callback func(peer *ml.Node, meta NodeMeta) error) *Events {
	dg.notifyAlive = callback

	return dg
}

func (dg *Events) NodeMeta(int) []byte {
	meta := dg.nodemeta()

	dg.Log().Debug().Interface("meta", meta).Msg("node meta requested")

	return dg.requestNodeMeta(meta)
}

func (dg *Events) SetNodeMeta(callback func(NodeMeta) []byte) *Events {
	dg.requestNodeMeta = callback

	return dg
}

func (dg *Events) NotifyMsg(b []byte) {
	dg.Log().Debug().Int("msg", len(b)).Msg("msg received")

	dg.notifyMsg(b)
}

func (dg *Events) SetNotifyMsg(callback func([]byte)) *Events {
	dg.notifyMsg = callback

	return dg
}

func (dg *Events) GetBroadcasts(_, _ int) [][]byte {
	dg.Log().Verbose().Msg("get Broadcast")

	return nil
}

func (dg *Events) LocalState(join bool) []byte {
	return dg.localState(join)
}

func (dg *Events) SetLocalState(callback func(bool) []byte) *Events {
	dg.localState = callback

	return dg
}

func (dg *Events) MergeRemoteState(buf []byte, join bool) {
	dg.Log().Debug().Interface("buf", buf).Bool("join", join).Msg("MergeRemoteState received")

	dg.mergeRemoteState(buf, join)
}

func (dg *Events) SetMergeRemoteState(callback func([]byte, bool)) *Events {
	dg.mergeRemoteState = callback

	return dg
}

func (dg *Events) updateNodeMeta(meta NodeMeta) *Events {
	dg.Lock()
	defer dg.Unlock()

	dg.meta = meta

	return dg
}

func (dg *Events) loadMeta(event string, peer *ml.Node) (NodeMeta, logging.Logger, error) {
	l := dg.Log().WithLogger(func(lctx logging.Context) logging.Emitter {
		return LogNode(lctx, peer).Str("event", event)
	})

	addr := peer.Address()

	meta, err := NewNodeMetaFromBytes(peer.Meta)
	if err != nil {
		err = xerrors.Errorf("wrong peer for %s: %w", event, err)
		l.Error().Err(err).Msg("failed to parse meta")

		return NodeMeta{}, l, err
	}

	u, uaddr, err := publishToAddress(meta.Publish())
	if err != nil {
		return NodeMeta{}, l, err
	}
	if meta.Publish().String() != u.String() {
		return NodeMeta{}, l, xerrors.Errorf("wrong publish url in meta, %q", meta.Publish().String())
	}
	if addr != uaddr {
		return NodeMeta{}, l, xerrors.Errorf("address does not match with peer, %q != %q", addr, uaddr)
	}

	l = l.WithLogger(func(lctx logging.Context) logging.Emitter {
		return LogNodeMeta(lctx, meta)
	})

	return meta, l, nil
}

func (dg *Events) nodemeta() NodeMeta {
	dg.RLock()
	defer dg.RUnlock()

	return dg.meta
}

func (dg *Events) setDiscoveryNotifyJoin(callback func(peer *ml.Node, meta NodeMeta) error) *Events {
	dg.discoveryNotifyJoin = callback

	return dg
}

func (dg *Events) setDiscoveryNotifyLeave(callback func(peer *ml.Node, meta NodeMeta) error) *Events {
	dg.discoveryNotifyLeave = callback

	return dg
}

func (dg *Events) setDiscoveryNotifyUpdate(callback func(peer *ml.Node, meta NodeMeta)) *Events {
	dg.discoveryNotifyUpdate = callback

	return dg
}

func (dg *Events) setDiscoveryNotifyMerge(callback func(peers []*ml.Node, metas map[string]NodeMeta) error) *Events {
	dg.discoveryNotifyMerge = callback

	return dg
}
