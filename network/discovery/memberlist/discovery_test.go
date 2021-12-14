package memberlist

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/launch"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type dummyNode struct {
	sync.RWMutex
	local     node.Local
	connInfo  network.ConnInfo
	node      string
	addr      string
	discovery *Discovery
}

type testDiscovery struct {
	suite.Suite
	BaseDiscoveryTest
	networkID base.NetworkID
	encs      *encoder.Encoders
	enc       encoder.Encoder
}

func (t *testDiscovery) SetupSuite() {
	t.networkID = base.NetworkID(util.UUID().Bytes())

	t.enc = jsonenc.NewEncoder()
	t.encs = encoder.NewEncoders()
	for _, e := range []encoder.Encoder{t.enc} {
		if err := t.encs.AddEncoder(e); err != nil {
			panic(err)
		}
	}

	for i := range launch.EncoderTypes {
		if err := t.encs.AddType(launch.EncoderTypes[i]); err != nil {
			panic(err)
		}
	}

	for i := range launch.EncoderHinters {
		if err := t.encs.AddHinter(launch.EncoderHinters[i]); err != nil {
			panic(err)
		}
	}

	if err := t.encs.Initialize(); err != nil {
		panic(err)
	}
}

func (t *testDiscovery) newNodes(n int) (map[string]*dummyNode, map[string]http.HandlerFunc) {
	publishes := map[string]http.HandlerFunc{}

	nodes := map[string]*dummyNode{}
	for i := 0; i < n; i++ {
		n := t.newNode(i, publishes)
		nodes[n.node] = n
		publishes[n.connInfo.URL().String()] = nil
	}

	return nodes, publishes
}

func (t *testDiscovery) newNode(i int, publishes map[string]http.HandlerFunc) *dummyNode {
	n := new(dummyNode)

	n.node = fmt.Sprintf("n%d", i)

	connInfo, err := network.NewHTTPConnInfoFromString(fmt.Sprintf("https://%s:443", n.node), false)
	t.NoError(err)
	n.connInfo = connInfo

	n.local = node.NewLocal(base.MustNewStringAddress(n.node), key.NewBasePrivatekey())

	_, n.addr, _ = publishToAddress(connInfo.URL())

	n.discovery = t.NewDiscovery(n.local, connInfo, t.networkID, t.enc, publishes)

	return n
}

func (t *testDiscovery) copyNode(
	orig *dummyNode,
	newurl string,
	publishes map[string]http.HandlerFunc,
) *dummyNode {
	connInfo, err := network.NewHTTPConnInfoFromString(newurl, false)
	t.NoError(err)

	t.False(orig.connInfo.Equal(connInfo))
	t.NotEqual(orig.connInfo.URL().String(), connInfo.URL().String())

	n := new(dummyNode)
	n.node = orig.node
	n.connInfo = connInfo

	n.local = node.NewLocal(orig.local.Address(), orig.local.Privatekey())

	_, n.addr, _ = publishToAddress(connInfo.URL())

	n.discovery = NewDiscovery(n.local, n.connInfo, t.networkID, t.enc)
	_ = n.discovery.SetRequest(orig.discovery.request)

	return n
}

func (t *testDiscovery) initializeNodes(nodes map[string]*dummyNode, publishes map[string]http.HandlerFunc) error {
	for _, n := range nodes {
		// NOTE uncomment for logging
		// l := logging.TestLogging.Log().With().Str("local", k).Logger()
		// _ = n.discovery.SetLogging(l)

		if err := n.discovery.Initialize(); err != nil {
			return err
		}

		publishes[n.connInfo.URL().String()] = n.discovery.Handler(nil)
	}

	return nil
}

func (t *testDiscovery) startNodes(nodes map[string]*dummyNode, joins map[string]*dummyNode) error {
	if joins == nil {
		joins = nodes
	}

	targets := make([]ConnInfo, len(joins))
	var i int
	for _, n := range joins {
		targets[i] = NewConnInfoWithConnInfo("", n.connInfo.(network.HTTPConnInfo))
		i++
	}

	for _, n := range nodes {
		if err := n.discovery.Start(); err != nil {
			return err
		}
	}

	errch := make(chan error, len(nodes))
	wk := util.NewDistributeWorker(context.Background(), int64(len(nodes)), errch)
	defer wk.Close()

	go func() {
		defer wk.Done()

		for _, n := range nodes {
			dis := n.discovery
			if err := wk.NewJob(func(context.Context, uint64) error {
				return dis.Join(targets, 2)
			}); err != nil {
				t.T().Logf("failed to run job: %q", err)

				return
			}
		}
	}()

	if err := wk.Wait(); err != nil {
		return err
	}

	close(errch)

	for err := range errch {
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *testDiscovery) stopNodes(nodes map[string]*dummyNode) error {
	for _, n := range nodes {
		if err := n.discovery.Stop(); err != nil {
			return err
		}
	}

	return nil
}

func (t *testDiscovery) checkJoinedNodes(nodes map[string]*dummyNode, excludes ...string /* dummyNode.addr */) {
	isExcluded := func(n *dummyNode) bool {
		for j := range excludes {
			if excludes[j] == n.addr {
				return true
			}
		}

		return false
	}

	addrs := make([]string, len(nodes)-len(excludes))
	urls := make([]string, len(nodes)-len(excludes))

	var i int
	for _, n := range nodes {
		if isExcluded(n) {
			continue
		}

		addrs[i] = n.addr
		urls[i] = n.connInfo.URL().String()
		i++
	}
	sort.Strings(addrs)
	sort.Strings(urls)

	for i := range nodes {
		a := nodes[i]
		if isExcluded(a) {
			continue
		}

		ms := a.discovery.Nodes()

		t.Equal(len(addrs), len(ms), a.node)
		t.Equal(len(urls), len(ms), a.node)

		uaddrs := make([]string, len(nodes)-len(excludes))
		uurls := make([]string, len(nodes)-len(excludes))
		for j := range ms {
			uaddrs[j] = ms[j].(NodeConnInfo).ConnInfo.Address
			uurls[j] = ms[j].(NodeConnInfo).ConnInfo.URL().String()
		}

		sort.Strings(uaddrs)
		sort.Strings(uurls)

		t.Equal(addrs, uaddrs)
		t.Equal(urls, uurls)
	}
}

func (t *testDiscovery) TestJoin() {
	nodes, publishes := t.newNodes(5)

	names := make([]string, len(nodes))
	var i int
	for _, n := range nodes {
		names[i] = n.node
		i++
	}
	sort.Strings(names)

	t.NoError(t.initializeNodes(nodes, publishes))
	t.NoError(t.startNodes(nodes, nil))
	defer func() {
		t.NoError(t.stopNodes(nodes))
	}()

	t.checkJoinedNodes(nodes)
}

func (t *testDiscovery) TestLeave() {
	nodes, publishes := t.newNodes(5)
	local := nodes["n0"]

	t.NoError(t.initializeNodes(nodes, publishes))

	lefts := new(sync.Map)
	var wgleave sync.WaitGroup
	wgleave.Add(len(nodes))
	for i := range nodes {
		n := nodes[i]

		_ = n.discovery.SetNotifyLeave(func(connInfo discovery.NodeConnInfo, nlefts []discovery.NodeConnInfo) {
			if connInfo.URL().String() != local.connInfo.URL().String() {
				return
			}

			if _, found := lefts.Load(n.node); found {
				return
			}
			if len(nlefts) > 0 {
				panic("left")
			}

			lefts.Store(n.node, true)

			wgleave.Done()
		})
	}

	t.NoError(t.startNodes(nodes, nil))
	defer func() {
		t.NoError(t.stopNodes(nodes))
	}()

	t.checkJoinedNodes(nodes)

	t.NoError(local.discovery.Leave(time.Second * 2))
	t.T().Log("local node left")

	wgleave.Wait()

	t.checkJoinedNodes(nodes, local.addr)

	// NOTE local is removed from the joined nodes of local
	localNodes := local.discovery.Nodes()
	for i := range localNodes {
		n := localNodes[i]
		t.False(local.local.Address().Equal(n.Node()))
	}
}

func (t *testDiscovery) TestOneNodeStopped() {
	nodes, publishes := t.newNodes(5)
	local := nodes["n0"]

	t.NoError(t.initializeNodes(nodes, publishes))

	lefts := new(sync.Map)
	var wgleave sync.WaitGroup
	wgleave.Add(len(nodes) - 1)
	for i := range nodes {
		n := nodes[i]
		n.discovery.conf.ProbeInterval = time.Second * 1

		_ = n.discovery.SetNotifyLeave(func(connInfo discovery.NodeConnInfo, _ []discovery.NodeConnInfo) {
			if connInfo.URL().String() != local.connInfo.URL().String() {
				return
			}

			if _, found := lefts.Load(n.node); found {
				return
			}
			lefts.Store(n.node, true)

			wgleave.Done()
		})
	}

	t.NoError(t.startNodes(nodes, nil))
	defer func() {
		_ = t.stopNodes(nodes)
	}()

	t.checkJoinedNodes(nodes)

	t.NoError(local.discovery.Stop())

	t.T().Log("local node stopped; wait leave")

	wgleave.Wait()

	t.checkJoinedNodes(nodes, local.addr)
}

func (t *testDiscovery) TestBroadcastOverUDPBufferSize() {
	nodes, publishes := t.newNodes(5)
	local := nodes["n0"]

	t.NoError(t.initializeNodes(nodes, publishes))

	msg := []byte(strings.Repeat("a", local.discovery.conf.UDPBufferSize+1))
	t.True(len(msg) > local.discovery.conf.UDPBufferSize)

	msgs := new(sync.Map)
	var wgb sync.WaitGroup
	wgb.Add(len(nodes) - 1)

	nodesch := make(chan string, len(nodes)-1)
	for i := range nodes {
		n := nodes[i]

		_ = n.discovery.Events().SetNotifyMsg(func(b []byte) {
			if bytes.Equal(msg, b) {
				defer wgb.Done()

				if _, found := msgs.Load(n.node); found {
					return
				}
				msgs.Store(n.node, true)

				nodesch <- n.node
			}
		})
	}

	t.NoError(t.startNodes(nodes, nil))
	defer func() {
		t.NoError(t.stopNodes(nodes))
	}()

	t.checkJoinedNodes(nodes)

	t.T().Logf("broadcast message; size=%d", len(msg))
	t.NoError(local.discovery.Broadcast(msg))

	wgb.Wait()
	close(nodesch)

	var received []string
	for i := range nodesch {
		received = append(received, i)
	}

	var expected []string
	for i := range nodes {
		if nodes[i].node == local.node {
			continue
		}

		expected = append(expected, nodes[i].node)
	}

	sort.Strings(received)
	sort.Strings(expected)
	t.Equal(expected, received)
}

func (t *testDiscovery) TestEmptyPublishNode() {
	nodes, publishes := t.newNodes(5)
	empty := nodes["n1"]

	t.NoError(t.initializeNodes(nodes, publishes))

	var err error
	empty.discovery.events.meta, err = NewFakeNodeMeta("", false)
	t.NoError(err)

	joined := new(sync.Map)
	var wg sync.WaitGroup
	wg.Add(len(nodes) - 1)
	for i := range nodes {
		n := nodes[i]
		n.discovery.conf.ProbeInterval = time.Second * 1

		_ = n.discovery.SetNotifyJoin(func(ci discovery.NodeConnInfo) {
			if ci.URL().String() == empty.connInfo.URL().String() {
				return
			}

			if _, found := joined.Load(ci.(NodeConnInfo).ConnInfo.Address); found {
				return
			}
			joined.Store(ci.(NodeConnInfo).ConnInfo.Address, true)

			wg.Done()
		})
	}

	t.NoError(t.startNodes(nodes, nil))
	defer func() {
		_ = t.stopNodes(nodes)
	}()

	wg.Wait()

	t.checkJoinedNodes(nodes, empty.addr)
}

func (t *testDiscovery) TestNewSameNodeWithNewPublishURL() {
	nodes, publishes := t.newNodes(2)
	orig := nodes["n0"]

	// NOTE set new publish url to n0
	n00 := t.copyNode(orig, fmt.Sprintf("https://%s:443/findme", orig.node), publishes)
	newnodes := map[string]*dummyNode{"new-n0": n00}

	all := map[string]*dummyNode{}
	for i := range nodes {
		all[i] = nodes[i]
	}
	for i := range newnodes {
		all[i] = newnodes[i]
	}

	t.NoError(t.initializeNodes(all, publishes))

	// NOTE start all nodes except new node
	var wg sync.WaitGroup
	wg.Add(len(nodes))

	joined := new(sync.Map)
	for i := range nodes {
		n := nodes[i]
		n.discovery.conf.ProbeInterval = time.Second * 1

		_ = n.discovery.SetNotifyJoin(func(discovery.NodeConnInfo) {
			if _, found := joined.Load(n.node); found {
				return
			}
			joined.Store(n.node, true)

			wg.Done()
		})
	}

	t.NoError(t.startNodes(nodes, nil))
	defer func() {
		_ = t.stopNodes(nodes)
	}()

	wg.Wait()

	t.checkJoinedNodes(nodes)

	// NOTE new node trying to join
	wg = sync.WaitGroup{}
	wg.Add(len(nodes))

	joined = new(sync.Map)
	for i := range nodes {
		n := nodes[i]
		_ = n.discovery.SetNotifyJoin(func(ci discovery.NodeConnInfo) {
			if ci.(NodeConnInfo).ConnInfo.Address != n00.addr {
				return
			}

			if _, found := joined.Load(n.addr); found {
				return
			}
			joined.Store(n.addr, true)

			wg.Done()
		})
	}

	t.NoError(t.startNodes(newnodes, nodes))
	defer func() {
		_ = t.stopNodes(newnodes)
	}()

	wg.Wait()

	// NOTE new node and orig are joined
	t.checkJoinedNodes(all)

	// NOTE orig node will leave
	wg = sync.WaitGroup{}
	wg.Add(len(all) - 1)

	lefts := new(sync.Map)
	for i := range all {
		n := all[i]
		if n.addr == orig.addr {
			continue
		}

		_ = n.discovery.SetNotifyLeave(func(connInfo discovery.NodeConnInfo, nlefts []discovery.NodeConnInfo) {
			if connInfo.(NodeConnInfo).ConnInfo.Address != orig.addr {
				return
			}

			if _, found := lefts.Load(n.node); found {
				return
			}

			if len(nlefts) < 1 {
				panic("empty")
			}

			lefts.Store(n.node, true)

			wg.Done()
		})
	}

	t.NoError(orig.discovery.Leave(time.Second * 2))
	t.T().Log("orig node left")

	wg.Wait()

	t.checkJoinedNodes(all, orig.addr)
}

func (t *testDiscovery) TestOverMaxNodeConns() {
	nodes, publishes := t.newNodes(2)
	orig := nodes["n0"]

	// NOTE set new publish url to n0
	n00 := t.copyNode(orig, fmt.Sprintf("https://%s:443/findme", orig.node), publishes)
	n01 := t.copyNode(orig, fmt.Sprintf("https://%s:443/showme", orig.node), publishes)
	newnodes := map[string]*dummyNode{"n00": n00}

	all := map[string]*dummyNode{}
	for i := range nodes {
		all[i] = nodes[i]
	}
	for i := range newnodes {
		all[i] = newnodes[i]
	}

	t.NoError(t.initializeNodes(all, publishes))

	n01s := map[string]*dummyNode{"n01": n01}
	t.NoError(t.initializeNodes(n01s, publishes))

	for k, n := range all {
		t.T().Logf("set MaxNodeConns=2 of %s", k)
		_ = n.discovery.SetMaxNodeConns(2)
	}

	// NOTE start all nodes except new node
	var wg sync.WaitGroup
	wg.Add(len(nodes))

	joined := new(sync.Map)
	for i := range nodes {
		n := nodes[i]
		n.discovery.conf.ProbeInterval = time.Second * 1

		_ = n.discovery.SetNotifyJoin(func(discovery.NodeConnInfo) {
			if _, found := joined.Load(n.node); found {
				return
			}
			joined.Store(n.node, true)

			wg.Done()
		})
	}

	t.NoError(t.startNodes(nodes, nil))
	defer func() {
		_ = t.stopNodes(nodes)
	}()

	wg.Wait()

	t.checkJoinedNodes(nodes)

	// NOTE new node trying to join
	wg = sync.WaitGroup{}
	wg.Add(len(nodes))

	joined = new(sync.Map)
	for i := range nodes {
		n := nodes[i]
		_ = n.discovery.SetNotifyJoin(func(ci discovery.NodeConnInfo) {
			if ci.(NodeConnInfo).ConnInfo.Address != n00.addr {
				return
			}

			if _, found := joined.Load(n.addr); found {
				return
			}
			joined.Store(n.addr, true)

			wg.Done()
		})
	}

	t.NoError(t.startNodes(newnodes, nodes))
	defer func() {
		_ = t.stopNodes(newnodes)
	}()

	wg.Wait()

	// NOTE new node and orig are joined
	t.checkJoinedNodes(all)

	// NOTE n01 trying to join, but it will be failed
	err := t.startNodes(n01s, all)
	t.Contains(err.Error(), "joining canceled; over max retry")
}

func (t *testDiscovery) TestJoiningDeclined() {
	nodes, publishes := t.newNodes(3)
	local := nodes["n0"]

	t.NoError(t.initializeNodes(nodes, publishes))

	for i := range nodes {
		n := nodes[i]
		if n.local.Address().Equal(local.local.Address()) {
			continue
		}

		// NOTE set to decline local
		handler := n.discovery.Handler(func(ms NodeMessage) error {
			if ms.node.Equal(local.local.Address()) {
				return JoinDeclinedError.Errorf("local declined by %q", n.local.Address())
			}

			return nil
		})
		publishes[n.connInfo.URL().String()] = handler
	}

	others := map[string]*dummyNode{}
	for k := range nodes {
		n := nodes[k]
		if n.node == local.node {
			continue
		}

		others[k] = n
	}

	joined := new(sync.Map)
	var wg sync.WaitGroup
	wg.Add(len(others))
	for i := range others {
		n := others[i]
		n.discovery.conf.ProbeInterval = time.Second * 1

		_ = n.discovery.SetNotifyJoin(func(connInfo discovery.NodeConnInfo) {
			if _, found := joined.Load(n.node); found {
				return
			}
			joined.Store(n.node, true)

			wg.Done()
		})
	}

	t.NoError(t.startNodes(others, nil))
	defer func() {
		t.NoError(t.stopNodes(nodes))
	}()

	wg.Wait()

	t.checkJoinedNodes(others)

	err := t.startNodes(map[string]*dummyNode{local.node: local}, others)
	t.True(errors.Is(err, JoiningCanceledError))
	t.Contains(err.Error(), "over max retry")
}

func (t *testDiscovery) TestDeclinedAfterJoin() {
	nodes, publishes := t.newNodes(3)
	local := nodes["n0"]

	t.NoError(t.initializeNodes(nodes, publishes))

	var left int64

	joined := new(sync.Map)
	var wgjoin sync.WaitGroup
	wgjoin.Add(len(nodes))
	for i := range nodes {
		n := nodes[i]
		n.discovery.conf.ProbeInterval = time.Second * 1

		_ = n.discovery.SetNotifyJoin(func(discovery.NodeConnInfo) {
			if _, found := joined.Load(n.node); found {
				return
			}
			joined.Store(n.node, true)

			wgjoin.Done()
		})

		handler := n.discovery.Handler(func(ms NodeMessage) error {
			if atomic.LoadInt64(&left) < 1 {
				return nil
			}

			if ms.node.Equal(local.local.Address()) {
				return JoinDeclinedError.Errorf("local declined by %q", n.local.Address())
			}

			return nil
		})
		publishes[n.connInfo.URL().String()] = handler
	}

	others := map[string]*dummyNode{}
	for k := range nodes {
		n := nodes[k]
		if n.node == local.node {
			continue
		}

		others[k] = n
	}

	lefts := new(sync.Map)
	var wgleave sync.WaitGroup
	wgleave.Add(len(others))
	for i := range others {
		n := others[i]

		_ = n.discovery.SetNotifyLeave(func(connInfo discovery.NodeConnInfo, _ []discovery.NodeConnInfo) {
			if connInfo.URL().String() != local.connInfo.URL().String() {
				return
			}

			if _, found := lefts.Load(n.node); found {
				return
			}
			lefts.Store(n.node, true)

			wgleave.Done()
		})
	}

	leftslocal := new(sync.Map)
	var wgleavelocal sync.WaitGroup
	wgleavelocal.Add(len(others))
	_ = local.discovery.SetNotifyLeave(func(connInfo discovery.NodeConnInfo, _ []discovery.NodeConnInfo) {
		if connInfo.URL().String() == local.connInfo.URL().String() {
			return
		}

		if _, found := leftslocal.Load(connInfo.Node()); found {
			return
		}
		leftslocal.Store(connInfo.Node(), true)

		wgleavelocal.Done()
	})

	t.NoError(t.startNodes(nodes, nil))
	defer func() {
		_ = t.stopNodes(nodes)
	}()

	wgjoin.Wait()

	t.checkJoinedNodes(nodes)

	t.T().Logf("nodes except local will not response the requests from local, %q", local.node)
	<-time.After(time.Second * 3)
	atomic.AddInt64(&left, 1)

	wgleave.Wait()

	wgleavelocal.Wait()

	t.Equal(1, len(local.discovery.Nodes()))
	leftlocal := local.discovery.Nodes()[0]
	t.True(leftlocal.Node().Equal(local.local.Address()))
}

func TestDiscovery(t *testing.T) {
	suite.Run(t, new(testDiscovery))
}
