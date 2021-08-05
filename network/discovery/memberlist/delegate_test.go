package memberlist

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/network/discovery"
	"golang.org/x/xerrors"
)

func (t *testDiscovery) nodepoolDelegate(
	local *dummyNode,
	nodes map[string]*dummyNode,
) (*discovery.NodepoolDelegate, *network.Nodepool, error) {
	ch, err := discovery.LoadNodeChannel(local.connInfo, t.encs, time.Second*5)
	if err != nil {
		return nil, nil, err
	}

	np := network.NewNodepool(local.local, ch)
	for i := range nodes {
		no := nodes[i]
		if no.local.Address().Equal(local.local.Address()) {
			continue
		}
		if err := np.Add(no.local, nil); err != nil {
			return nil, nil, err
		}
	}

	dg := discovery.NewNodepoolDelegate(np, t.encs, time.Second*5)
	// dg.SetLogging(logging.TestLogging)

	return dg, np, nil
}

func (t *testDiscovery) TestNotifyJoin() {
	nodes, publishes := t.newNodes(2)
	local := nodes["n0"]

	dg, np, err := t.nodepoolDelegate(local, nodes)
	t.NoError(err)

	_ = local.discovery.SetNotifyJoin(dg.NotifyJoin)

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

	for i := range nodes {
		no := nodes[i]
		if no.local.Address().Equal(local.local.Address()) {
			continue
		}

		t.NoError(t.checkNodeJoinedNodepool(np, no.local.Address()))
		_, ch, _ := np.Node(no.local.Address())

		t.Equal(no.connInfo.URL().String(), ch.ConnInfo().URL().String())
		t.Equal(no.connInfo.Insecure(), ch.ConnInfo().Insecure())
	}
}

func (t *testDiscovery) TestNotifyLeave() {
	nodes, publishes := t.newNodes(5)
	local := nodes["n0"]
	remote := nodes["n1"]

	dg, np, err := t.nodepoolDelegate(local, nodes)
	t.NoError(err)

	_ = local.discovery.SetNotifyJoin(dg.NotifyJoin)

	t.NoError(t.initializeNodes(nodes, publishes))

	lefts := new(sync.Map)
	var wgleave sync.WaitGroup
	wgleave.Add(len(nodes))
	for i := range nodes {
		n := nodes[i]

		_ = n.discovery.SetNotifyLeave(func(connInfo discovery.NodeConnInfo, nlefts []discovery.NodeConnInfo) {
			if n.addr == local.addr {
				dg.NotifyLeave(connInfo, nlefts)
			}

			if connInfo.URL().String() != remote.connInfo.URL().String() {
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

	t.NoError(t.checkNodeJoinedNodepool(np, remote.local.Address()))
	_, ch, _ := np.Node(remote.local.Address())

	t.Equal(remote.connInfo.URL().String(), ch.ConnInfo().URL().String())
	t.Equal(remote.connInfo.Insecure(), ch.ConnInfo().Insecure())

	t.NoError(remote.discovery.Leave(time.Second * 2))
	t.T().Log("remote node left")

	wgleave.Wait()

	t.checkJoinedNodes(nodes, remote.addr)

	t.NoError(t.checkNodeLeftNodepool(np, remote.local.Address()))
}

func (t *testDiscovery) TestNotifyLeaveWithLefts() {
	nodes, publishes := t.newNodes(3)
	local := nodes["n0"]
	remote := nodes["n1"]

	dg, np, err := t.nodepoolDelegate(local, nodes)
	t.NoError(err)

	// NOTE set new publish url to n0
	nremote := t.copyNode(remote, fmt.Sprintf("https://%s:443/findme", remote.node), publishes)
	newnodes := map[string]*dummyNode{"new-n1": nremote}

	all := map[string]*dummyNode{}
	for i := range nodes {
		all[i] = nodes[i]
	}
	for i := range newnodes {
		all[i] = newnodes[i]
	}

	t.NoError(t.initializeNodes(all, publishes))

	// NOTE start all nodes except new remote
	var wg sync.WaitGroup
	wg.Add(len(nodes))

	joined := new(sync.Map)
	for i := range nodes {
		n := nodes[i]
		n.discovery.conf.ProbeInterval = time.Second * 1

		_ = n.discovery.SetNotifyJoin(func(ci discovery.NodeConnInfo) {
			if n.addr == local.addr {
				dg.NotifyJoin(ci)
			}

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

	t.NoError(t.checkNodeJoinedNodepool(np, remote.local.Address()))
	_, ch, _ := np.Node(remote.local.Address())

	t.Equal(remote.connInfo.URL().String(), ch.ConnInfo().URL().String())
	t.Equal(remote.connInfo.Insecure(), ch.ConnInfo().Insecure())

	// NOTE new node trying to join
	wg = sync.WaitGroup{}
	wg.Add(len(nodes))

	joined = new(sync.Map)
	for i := range nodes {
		n := nodes[i]
		_ = n.discovery.SetNotifyJoin(func(ci discovery.NodeConnInfo) {
			if n.addr == local.addr {
				go dg.NotifyJoin(ci)
			}

			if ci.(NodeConnInfo).ConnInfo.Address != nremote.addr {
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

	// NOTE new node and remote are joined
	t.checkJoinedNodes(all)

	t.NoError(t.checkNodeJoinedNodepool(np, nremote.local.Address()))
	_, ch, _ = np.Node(nremote.local.Address())

	t.Equal(nremote.connInfo.URL().String(), ch.ConnInfo().URL().String())
	t.Equal(nremote.connInfo.Insecure(), ch.ConnInfo().Insecure())

	// NOTE remote node will leave
	wg = sync.WaitGroup{}
	wg.Add(len(all) - 1)

	lefts := new(sync.Map)
	for i := range all {
		n := all[i]
		if n.addr == remote.addr {
			continue
		}

		_ = n.discovery.SetNotifyLeave(func(connInfo discovery.NodeConnInfo, nlefts []discovery.NodeConnInfo) {
			if n.addr == local.addr {
				dg.NotifyLeave(connInfo, nlefts)
			}

			if connInfo.(NodeConnInfo).ConnInfo.Address != remote.addr {
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

	t.NoError(remote.discovery.Leave(time.Second * 2))
	t.T().Log("remote node left")

	wg.Wait()

	t.checkJoinedNodes(all, remote.addr)

	t.NoError(t.checkNodeJoinedNodepool(np, nremote.local.Address()))

	t.Equal(nremote.connInfo.URL().String(), ch.ConnInfo().URL().String())
	t.Equal(nremote.connInfo.Insecure(), ch.ConnInfo().Insecure())
}

func (t *testDiscovery) checkNodeLeftNodepool(nodepool *network.Nodepool, no base.Address) error {
	for {
		select {
		case <-time.After(time.Second * 3):
			return xerrors.Errorf("expired")
		default:
			_, ch, found := nodepool.Node(no)
			t.True(found)

			if ch == nil {
				return nil
			}

			<-time.After(time.Second)
		}
	}
}

func (t *testDiscovery) checkNodeJoinedNodepool(nodepool *network.Nodepool, no base.Address) error {
	for {
		select {
		case <-time.After(time.Second * 3):
			return xerrors.Errorf("expired")
		default:
			_, ch, found := nodepool.Node(no)
			t.True(found)

			if ch != nil {
				return nil
			}

			<-time.After(time.Second)
		}
	}
}
