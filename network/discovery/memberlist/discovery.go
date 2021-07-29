package memberlist

import (
	"context"
	"io"
	stdlog "log"
	"net"
	"net/http"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/bluele/gcache"
	ml "github.com/hashicorp/memberlist"
	"github.com/lucas-clemente/quic-go"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/semaphore"
	"golang.org/x/xerrors"
)

var (
	DefaultDiscoveryPath      = "/_join"
	defaultConfig             = ml.DefaultWANConfig()
	defaultTCPTimeout         = time.Second * 3
	defaultProbeInterval      = defaultConfig.ProbeInterval
	defaultProbeTimeout       = defaultConfig.ProbeTimeout
	DefaultMaxNodeConns  uint = 2
)

type Discovery struct {
	*logging.Logging
	*util.ContextDaemon
	local            *node.Local
	networkID        base.NetworkID
	enc              encoder.Encoder
	tcpTimeout       time.Duration
	probeInterval    time.Duration
	probeTimeout     time.Duration
	request          QuicRequest
	ma               *ConnInfoMap
	networkConnInfo  network.ConnInfo
	connInfo         ConnInfo
	ml               *ml.Memberlist
	events           *Events
	conf             *ml.Config
	nodesLock        sync.RWMutex
	nodes            []discovery.NodeConnInfo
	nodeConnInfos    map[ /* peer address */ string]NodeConnInfo
	nodesByAddr      map[ /* node address */ string][] /* peer address */ string
	lameNodes        *cache.GCache
	nodeCache        *cache.GCache
	notifyLock       sync.RWMutex
	notifyJoinFunc   func(discovery.NodeConnInfo)
	notifyLeaveFunc  func(discovery.NodeConnInfo, []discovery.NodeConnInfo /* left */)
	notifyUpdateFunc func(discovery.NodeConnInfo)
	checkMessage     func(NodeMessage) error
	maxNodeConns     int
}

func NewDiscovery(
	local *node.Local,
	connInfo network.ConnInfo,
	networkID base.NetworkID,
	enc encoder.Encoder,
) *Discovery {
	dis := &Discovery{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "memberlist-discovery")
		}),
		local:            local,
		networkConnInfo:  connInfo,
		networkID:        networkID,
		enc:              enc,
		tcpTimeout:       defaultTCPTimeout,
		probeInterval:    defaultProbeInterval,
		probeTimeout:     defaultProbeTimeout,
		request:          DefaultRequest(DefaultDiscoveryPath),
		nodeConnInfos:    map[string]NodeConnInfo{},
		nodesByAddr:      map[string][]string{},
		notifyJoinFunc:   func(discovery.NodeConnInfo) {},
		notifyLeaveFunc:  func(discovery.NodeConnInfo, []discovery.NodeConnInfo) {},
		notifyUpdateFunc: func(discovery.NodeConnInfo) {},
		checkMessage:     func(NodeMessage) error { return nil },
		maxNodeConns:     int(DefaultMaxNodeConns),
	}
	dis.ContextDaemon = util.NewContextDaemon("memberlist-discovery", dis.run)

	return dis
}

func (dis *Discovery) SetLogger(l logging.Logger) logging.Logger {
	if dis.events != nil {
		_ = dis.events.SetLogger(l)
	}

	if dis.conf != nil && dis.conf.Transport != nil {
		_ = dis.conf.Transport.(*QuicTransport).SetLogger(l)
	}

	_ = dis.ContextDaemon.SetLogger(l)

	return dis.Logging.SetLogger(l)
}

func (dis *Discovery) SetRequest(f QuicRequest) *Discovery {
	dis.request = f

	return dis
}

func (dis *Discovery) SetTimeout(tcpTimeout, probeInterval, probeTimeout time.Duration) *Discovery {
	dis.tcpTimeout = tcpTimeout
	dis.probeInterval = probeInterval
	dis.probeTimeout = probeTimeout

	return dis
}

func (dis *Discovery) SetMaxNodeConns(i uint) *Discovery {
	if i < DefaultMaxNodeConns {
		dis.Log().Warn().
			Uint("max", i).
			Uint("default", DefaultMaxNodeConns).
			Msg("too small number for max node conns; ignore")

		return dis
	}

	dis.maxNodeConns = int(i)

	return dis
}

func (dis *Discovery) Initialize() error {
	dis.ma = NewConnInfoMap()

	dis.connInfo, _ = dis.ma.add(dis.networkConnInfo.URL(), dis.networkConnInfo.Insecure())

	if dis.tcpTimeout < 1 {
		dis.tcpTimeout = defaultTCPTimeout
	}
	if dis.probeInterval < 1 {
		dis.probeInterval = defaultProbeInterval
	}
	if dis.probeTimeout < 1 {
		dis.probeTimeout = defaultProbeTimeout
	}

	var err error
	dis.nodeCache, err = cache.NewGCache("lru", 1<<16, dis.probeInterval*2)
	if err != nil {
		return err
	}

	dis.lameNodes, err = cache.NewGCache("lru", 1<<16, dis.probeInterval*2)
	if err != nil {
		return err
	}

	meta, err := NewNodeMeta(dis.connInfo.URL().String(), dis.connInfo.Insecure())
	if err != nil {
		return err
	}

	dis.events = NewEvents(dis.connInfo, meta, dis.ma)
	_ = dis.events.SetLogger(dis.Log())

	_ = dis.events.setDiscoveryNotifyJoin(dis.whenJoined)
	_ = dis.events.setDiscoveryNotifyLeave(dis.whenLeave)
	_ = dis.events.setDiscoveryNotifyUpdate(dis.whenUpdated)
	_ = dis.events.setDiscoveryNotifyMerge(dis.whenMerged)

	conf, err := dis.createConfig()
	if err != nil {
		return err
	}
	dis.conf = conf

	return nil
}

func (dis *Discovery) Start() error {
	i, err := ml.Create(dis.conf)
	if err != nil {
		return err
	}

	dis.ml = i

	return dis.ContextDaemon.Start()
}

func (dis *Discovery) run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	transport := dis.conf.Transport.(*QuicTransport)
	go func() {
		for range ticker.C {
			dis.cleanOldConnInfo(time.Minute * 3)
			transport.checkConnections()
		}
	}()

	<-ctx.Done()
	err := dis.ml.Shutdown()
	if err != nil {
		dis.Log().Error().Err(err).Msg("failed to stop")
	}

	return err
}

func (dis *Discovery) GracefuleStop(timeout time.Duration) error {
	if err := dis.Leave(timeout); err != nil {
		dis.Log().Error().Err(err).Msg("failed to leave")
	}

	return dis.Stop()
}

func (dis *Discovery) NodeMeta() NodeMeta {
	return dis.events.meta
}

func (dis *Discovery) LenNodes() int {
	dis.nodesLock.RLock()
	defer dis.nodesLock.RUnlock()

	return len(dis.nodeConnInfos)
}

func (dis *Discovery) Nodes() []discovery.NodeConnInfo {
	dis.nodesLock.RLock()
	defer dis.nodesLock.RUnlock()

	if dis.nodes != nil {
		return dis.nodes
	}

	nodes := make([]discovery.NodeConnInfo, len(dis.nodeConnInfos))

	// NOTE []discovery.NodeConnInfo keeps the added order in one node
	var i int
	for j := range dis.nodesByAddr {
		for k := range dis.nodesByAddr[j] {
			nodes[i] = dis.nodeConnInfos[dis.nodesByAddr[j][k]]
			i++
		}
	}

	dis.nodes = nodes

	return dis.nodes
}

func (dis *Discovery) Handler(callback func(NodeMessage) error) http.HandlerFunc {
	if callback == nil {
		callback = func(NodeMessage) error { return nil }
	}

	return dis.conf.Transport.(*QuicTransport).handler(func(ms NodeMessage) error {
		return dis.transportHandler(ms, callback)
	})
}

func (dis *Discovery) Join(nodes []ConnInfo, maxretry int) error {
	var filtered []ConnInfo // nolint:prealloc
	founds := map[string]bool{}
	for i := range nodes {
		connInfo := nodes[i]
		if connInfo.Address == dis.connInfo.Address {
			continue
		}

		if _, found := founds[connInfo.URL().String()]; found {
			continue
		}

		filtered = append(filtered, connInfo)
	}

	if len(filtered) < 1 {
		dis.Log().Debug().Msg("empty joining nodes")

		return nil
	}

	if maxretry < 0 {
		dis.Log().Debug().Int("maxretry", maxretry).Msg("max retry is under 0, keep retry")
	}

	ticker := time.NewTicker(time.Second * 3)
	defer ticker.Stop()

	retry := 1
	for range ticker.C {
		i, err := dis.join(filtered)
		if err == nil {
			break
		}

		if i > 0 && i != len(filtered) {
			break
		}

		if maxretry >= 0 && retry >= maxretry {
			return JoiningCanceledError.Errorf("over max retry, %d", maxretry)
		}

		dis.Log().Error().Err(err).Int("retry", retry).Msg("failed to join; will retry")

		if maxretry >= 0 {
			retry++
		}
	}

	dis.Log().Debug().Msg("joined")

	return dis.UpdateNode(nil)
}

func (dis *Discovery) UpdateNode(meta *NodeMeta) error {
	if meta != nil {
		_ = dis.events.updateNodeMeta(*meta)
	}

	if err := dis.ml.UpdateNode(0); err != nil {
		dis.Log().Error().Err(err).Msg("failed to update node")

		return err
	}
	dis.Log().Debug().Msg("node requested to update node")

	return nil
}

func (dis *Discovery) notifyJoin() func(discovery.NodeConnInfo) {
	dis.notifyLock.RLock()
	defer dis.notifyLock.RUnlock()

	return dis.notifyJoinFunc
}

func (dis *Discovery) SetNotifyJoin(callback func(discovery.NodeConnInfo)) discovery.Discovery {
	dis.notifyLock.Lock()
	defer dis.notifyLock.Unlock()

	dis.notifyJoinFunc = callback

	return dis
}

func (dis *Discovery) notifyLeave() func(discovery.NodeConnInfo, []discovery.NodeConnInfo) {
	dis.notifyLock.RLock()
	defer dis.notifyLock.RUnlock()

	return dis.notifyLeaveFunc
}

func (dis *Discovery) SetNotifyLeave(
	callback func(discovery.NodeConnInfo, []discovery.NodeConnInfo),
) discovery.Discovery {
	dis.notifyLock.Lock()
	defer dis.notifyLock.Unlock()

	dis.notifyLeaveFunc = callback

	return dis
}

func (dis *Discovery) notifyUpdate() func(discovery.NodeConnInfo) {
	dis.notifyLock.RLock()
	defer dis.notifyLock.RUnlock()

	return dis.notifyUpdateFunc
}

func (dis *Discovery) SetNotifyUpdate(callback func(discovery.NodeConnInfo)) discovery.Discovery {
	dis.notifyLock.Lock()
	defer dis.notifyLock.Unlock()

	dis.notifyUpdateFunc = callback

	return dis
}

func (dis *Discovery) SetCheckMessage(callback func(NodeMessage) error) *Discovery {
	dis.checkMessage = callback

	return dis
}

func (dis *Discovery) join(nodes []ConnInfo) (int, error) {
	if len(nodes) < 1 {
		dis.Log().Debug().Msg("empty nodes")

		return 0, nil
	}

	var addrs []string // nolint:prealloc
	for i := range nodes {
		j := nodes[i]
		if j.URL().String() == dis.connInfo.URL().String() {
			dis.Log().Warn().
				Interface("local", dis.local).
				Str("publish", j.URL().String()).
				Msg("same node found with local")

			continue
		}

		ci, _ := dis.ma.add(j.URL(), j.Insecure())
		addrs = append(addrs, ci.Address)
	}

	n, err := dis.ml.Join(addrs)
	if n < 1 {
		dis.Log().Warn().Err(err).Msg("failed to join network")
	} else {
		dis.Log().Info().Err(err).Int("nodes", n).Msg("node joined")
	}

	return n, err
}

func (dis *Discovery) Leave(timeout time.Duration) error {
	return dis.ml.Leave(timeout)
}

func (dis *Discovery) Events() *Events {
	return dis.events
}

// Broadcast send bytese message to joined nodes, except local node.
func (dis *Discovery) Broadcast(b []byte) error {
	sem := semaphore.NewWeighted(100)
	eg, ctx := errgroup.WithContext(context.Background())

	nodes := dis.Nodes()
	for i := range nodes {
		connInfo := nodes[i].(NodeConnInfo).ConnInfo
		if connInfo.Address == dis.connInfo.Address {
			continue
		}

		if err := sem.Acquire(ctx, 1); err != nil {
			return err
		}

		raddr, err := net.ResolveTCPAddr("tcp", connInfo.Address)
		if err != nil {
			return err
		}

		no := &ml.Node{Addr: raddr.IP, Port: uint16(raddr.Port)}

		eg.Go(func() error {
			defer sem.Release(1)

			return dis.ml.SendBestEffort(no, b)
		})
	}

	if err := sem.Acquire(ctx, 100); err != nil {
		if !xerrors.Is(err, context.Canceled) {
			return err
		}
	}

	return eg.Wait()
}

func (dis *Discovery) createConfig() (*ml.Config, error) {
	host, port, err := parseHostPort(dis.connInfo.Address)
	if err != nil {
		return nil, xerrors.Errorf("wrong advertise address: %w", err)
	}

	conf := ml.DefaultWANConfig()
	conf.Name = dis.connInfo.Address

	// TODO fine-tuning
	// memberlistConf.GossipInterval = 200 * time.Millisecond
	// memberlistConf.PushPullInterval = time.Second
	conf.AdvertiseAddr = host
	conf.AdvertisePort = port
	conf.TCPTimeout = dis.tcpTimeout

	conf.Merge = dis.events
	conf.Alive = dis.events
	conf.Events = dis.events
	conf.Delegate = dis.events

	// NOTE too narrow probe interval makes node to misconceives valid node.
	conf.ProbeInterval = dis.probeInterval
	conf.ProbeTimeout = dis.probeTimeout

	conf.IndirectChecks = 100
	conf.DisableTcpPings = true

	if dis.Log().Level() == zerolog.DebugLevel {
		conf.Logger = stdlog.New(
			logging.NewZerologSTDLoggingWriter(func() *zerolog.Event {
				return dis.Log().Debug().Event.Str("module", "hashicorp/memberlist")
			}), "", 0)
	} else {
		conf.LogOutput = io.Discard
	}

	tp := NewQuicTransport(
		dis.request,
		dis.newNodeMessage,
		dis.loadNodeMessage,
		dis.ma,
		conf.TCPTimeout,
	)
	_ = tp.SetLogger(dis.Log())

	conf.Transport = tp

	return conf, nil
}

func (dis *Discovery) newNodeMessage(body []byte, conid string) ([]byte, error) {
	ms := NewNodeMessage(
		dis.local.Address(),
		dis.connInfo,
		body,
		conid,
	)

	if err := ms.sign(dis.local.Privatekey(), dis.networkID); err != nil {
		return nil, err
	}

	return dis.enc.Marshal(ms)
}

func (dis *Discovery) loadNodeMessage(b []byte) (NodeMessage, error) {
	var ms NodeMessage
	if err := ms.Unpack(b, dis.enc); err != nil {
		return ms, xerrors.Errorf("failed to unmarshal NodeMessage: %w", err)
	}

	if err := ms.IsValid(dis.networkID); err != nil {
		return ms, err
	}

	if localtime.UTCNow().After(ms.signedAt.Add(dis.tcpTimeout * 2)) {
		return ms, xerrors.Errorf("too old node message received from %q(%q)", ms.node, ms.signedAt)
	}

	return ms, nil
}

func (dis *Discovery) cleanOldConnInfo(d time.Duration) {
	now := localtime.UTCNow()

	dis.ma.traverse(func(connInfo ConnInfo) bool {
		if connInfo.Address == dis.connInfo.Address {
			return true
		}

		if now.After(connInfo.LastActivated.Add(d)) {
			_ = dis.ma.remove(connInfo.Address)

			dis.Log().Debug().
				Str("address", connInfo.Address).
				Str("publish", connInfo.URL().String()).
				Msg("old ConnInfo removed")
		}

		return true
	})
}

func (dis *Discovery) whenJoined(peer *ml.Node, meta NodeMeta) error {
	dis.nodesLock.Lock()
	defer dis.nodesLock.Unlock()

	nci, isnew, err := dis.joinByPeer(peer, meta)
	if err != nil {
		return err
	}
	if isnew {
		dis.notifyJoin()(nci)
	}

	return nil
}

func (dis *Discovery) whenLeave(peer *ml.Node, _ NodeMeta) error {
	dis.nodesLock.Lock()
	defer dis.nodesLock.Unlock()

	addr := peer.Address()

	nci, found := dis.nodeConnInfos[addr]
	if !found {
		return nil
	}

	delete(dis.nodeConnInfos, addr)

	var lefts []discovery.NodeConnInfo
	if no := nci.Node().String(); len(dis.nodesByAddr[no]) < 2 {
		delete(dis.nodesByAddr, no)
	} else {
		lefts = make([]discovery.NodeConnInfo, len(dis.nodesByAddr[no])-1)
		n := make([]string, len(dis.nodesByAddr[no])-1)

		var j int
		for i := range dis.nodesByAddr[no] {
			aaddr := dis.nodesByAddr[no][i]
			if aaddr == addr {
				continue
			}

			n[j] = aaddr
			lefts[j] = dis.nodeConnInfos[aaddr]
			j++
		}

		dis.nodesByAddr[no] = n
	}

	dis.nodes = nil

	dis.Log().Debug().
		Str("node_address", nci.Node().String()).
		Str("peer", nci.ConnInfo.Address).
		Str("publish", nci.ConnInfo.URL().String()).
		Bool("insecure", nci.ConnInfo.Insecure()).
		Interface("lefts", lefts).
		Msg("node removed")

	_ = dis.nodeCache.Remove(addr)
	_ = dis.lameNodes.Remove(addr)

	dis.notifyLeave()(nci, lefts)

	return nil
}

func (dis *Discovery) whenUpdated(peer *ml.Node, meta NodeMeta) {
	addr := peer.Address()
	if addr == dis.connInfo.Address {
		return
	}

	dis.nodesLock.Lock()
	defer dis.nodesLock.Unlock()

	l := LogNodeMeta(LogNode(dis.Log().Debug(), peer), meta)

	i, err := dis.nodeCache.Get(addr)
	if err != nil {
		if !xerrors.Is(err, gcache.KeyNotFoundError) {
			LogNodeMeta(LogNode(dis.Log().Error(), peer), meta).
				Err(err).Msg("failed to get NodeMessage from node cache")
		} else {
			l.Msg("address not found in node cache; will be added to lame nodes")

			_ = dis.lameNodes.Set(addr, struct{}{}, 0)
		}

		return
	}

	ms := i.(NodeMessage)
	nci, isnew := dis.addNode(ms.node, addr, ms.URL(), ms.Insecure())
	if !isnew {
		return
	}

	dis.notifyUpdate()(nci)

	l.Msg("node updated")
}

func (dis *Discovery) whenMerged(peers []*ml.Node, metas map[string]NodeMeta) error {
	dis.nodesLock.Lock()
	defer dis.nodesLock.Unlock()

	for i := range peers {
		peer := peers[i]
		addr := peer.Address()
		meta, found := metas[addr]
		if !found {
			return xerrors.Errorf("meta missed of peer, %q", addr)
		}

		nci, isnew, err := dis.joinByPeer(peer, meta)
		if err != nil {
			return err
		}
		if isnew {
			dis.notifyJoin()(nci)
		}
	}

	return nil
}

func (dis *Discovery) updateMissingJoinedNode(ms NodeMessage) {
	dis.nodesLock.Lock()
	defer dis.nodesLock.Unlock()

	defer func() {
		_ = dis.nodeCache.Set(ms.Address, ms, 0)
	}()

	_, err := dis.lameNodes.Get(ms.Address)
	if err == nil {
		nci, isnew := dis.addNode(ms.node, ms.Address, ms.URL(), ms.Insecure())
		if isnew {
			dis.notifyJoin()(nci)
		}

		return
	}

	if !xerrors.Is(err, gcache.KeyNotFoundError) {
		dis.Log().Error().Err(err).Msg("failed to get NodeMessage from lame nodes cache")

		return
	}
}

func (dis *Discovery) joinByPeer(peer *ml.Node, meta NodeMeta) (NodeConnInfo, bool, error) {
	addr := peer.Address()
	if addr == dis.connInfo.Address {
		if local := dis.connInfo.URL().String(); meta.Publish().String() != local {
			err := xerrors.Errorf("weird joined node found; same address with local, but incorrect")

			LogNodeMeta(LogNode(dis.Log().Warn(), peer), meta).Str("local", local).
				Err(err).
				Msg("weird joined node found; same address with local, but incorrect")

			return NodeConnInfo{}, false, err
		}

		nci, isnew := dis.addNode(dis.local.Address(), addr, dis.connInfo.URL(), dis.connInfo.Insecure())

		return nci, isnew, nil
	}

	i, err := dis.nodeCache.Get(addr)
	if err == nil {
		ms := i.(NodeMessage)
		nci, isnew := dis.addNode(ms.node, addr, ms.URL(), ms.Insecure())

		return nci, isnew, nil
	}

	if xerrors.Is(err, gcache.KeyNotFoundError) {
		LogNodeMeta(LogNode(dis.Log().Debug(), peer), meta).
			Msg("address not found in node cache; will be added to lame nodes")

		_ = dis.lameNodes.Set(addr, struct{}{}, 0)

		return NodeConnInfo{}, false, nil
	}

	return NodeConnInfo{}, false, err
}

func (dis *Discovery) addNode(no base.Address, addr string, u *url.URL, insecure bool) (NodeConnInfo, bool) {
	defer func() {
		_ = dis.lameNodes.Remove(addr)
	}()

	if old, found := dis.nodeConnInfos[addr]; found {
		var update bool
		for _, f := range []func() bool{
			func() bool { return !old.Node().Equal(no) },
			func() bool { return old.ConnInfo.Address != addr },
			func() bool { return old.ConnInfo.URL().String() != u.String() },
			func() bool { return old.ConnInfo.Insecure() != insecure },
		} {
			if f() {
				update = true
				break
			}
		}

		if !update {
			return old, false
		}
	}

	connInfo := NewConnInfo(addr, u, insecure)

	nci := NewNodeConnInfo(connInfo, no)
	dis.nodeConnInfos[addr] = nci
	dis.nodesByAddr[no.String()] = append(dis.nodesByAddr[no.String()], addr)
	dis.nodes = nil

	dis.Log().Debug().
		Str("node_address", no.String()).
		Str("peer", addr).
		Str("publish", u.String()).
		Bool("insecure", insecure).
		Msg("new node added")

	return nci, true
}

func (dis *Discovery) transportHandler(ms NodeMessage, callback func(NodeMessage) error) error {
	if err := dis.checkMessage(ms); err != nil {
		return err
	}

	if err := dis.checkNodeConns(ms); err != nil {
		return err
	}

	if err := callback(ms); err != nil {
		return err
	}

	go dis.updateMissingJoinedNode(ms)

	return nil
}

// checkNodeConns checks how many connections are done by node, so over
// maxNodeConns, NodeMessage will be declined.
func (dis *Discovery) checkNodeConns(ms NodeMessage) error {
	dis.nodesLock.RLock()
	defer dis.nodesLock.RUnlock()

	if _, found := dis.nodeConnInfos[ms.ConnInfo.Address]; found {
		return nil
	}

	i, found := dis.nodesByAddr[ms.node.String()]
	if !found {
		return nil
	}

	// NOTE check connections conns by node
	if n := len(i); n >= dis.maxNodeConns {
		return JoinDeclinedError.Errorf("node already has many connection, %d", n)
	}

	return nil
}

func DefaultRequest(p string) QuicRequest {
	return func(
		ctx context.Context,
		insecure bool,
		timeout time.Duration,
		u, /* url */
		method string,
		body []byte,
		header http.Header,
	) (*http.Response, func() error, error) {
		var i *url.URL
		i, err := url.Parse(u)
		if err != nil {
			return nil, nil, err
		}
		i.Path = path.Join(i.Path, p)

		quicConfig := &quic.Config{HandshakeIdleTimeout: timeout}
		client, _ := quicnetwork.NewQuicClient(insecure, quicConfig)

		return client.Request(
			ctx,
			timeout,
			i.String(),
			method,
			body,
			header,
		)
	}
}
