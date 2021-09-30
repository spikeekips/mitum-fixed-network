package basicstates

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testHandover struct {
	baseTestState
}

func (t *testHandover) newHandover(local *isaac.Local, suffrage base.Suffrage) *Handover {
	return NewHandover(
		local.Channel().ConnInfo(),
		t.Encs,
		local.Policy(),
		local.Nodes(),
		suffrage,
	)
}

func (t *testHandover) TestEmptyDiscoveryURL() {
	_, err := NewHandoverWithDiscoveryURL(
		t.local.Channel().ConnInfo(),
		t.Encs,
		t.local.Policy(),
		t.local.Nodes(),
		t.Suffrage(t.local),
		nil,
	)
	t.NoError(err)
}

func (t *testHandover) TestStartNotSuffrage() {
	hd := t.newHandover(t.local, t.Suffrage(t.remote))

	hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
		return nil, nil, nil
	}

	t.NoError(hd.Start())
	defer func() {
		_ = hd.Stop()
	}()

	t.False(hd.UnderHandover())
	t.Nil(hd.OldNode())
	t.False(hd.IsReady())
}

func (t *testHandover) TestStartNotUnderhandover() {
	hd := t.newHandover(t.local, t.Suffrage(t.local))

	old := t.newChannel("https://old")
	hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
		return old, network.NodeInfoV0{}, nil
	}

	t.NoError(hd.Start())
	defer func() {
		t.NoError(hd.Stop())
	}()

	t.True(hd.UnderHandover())
	t.False(hd.IsReady())

	uold := hd.OldNode()
	t.NotNil(uold)
	t.True(old.ConnInfo().Equal(uold.ConnInfo()))
}

func (t *testHandover) TestStop() {
	hd := t.newHandover(t.local, t.Suffrage(t.local))

	old := t.newChannel("https://old")
	hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
		return old, network.NodeInfoV0{}, nil
	}

	t.NoError(hd.Start())
	defer func() {
		_ = hd.Stop()
	}()

	t.NoError(hd.Stop())

	t.False(hd.UnderHandover())
	t.Nil(hd.OldNode())
	t.False(hd.IsReady())
}

func (t *testHandover) TestStartAndRefresh() {
	hd := t.newHandover(t.local, t.Suffrage(t.local))

	old := t.newChannel("https://old")
	hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
		return old, network.NodeInfoV0{}, nil
	}

	t.NoError(hd.Start())
	defer func() {
		t.NoError(hd.Stop())
	}()

	t.NoError(hd.Refresh())

	t.True(hd.UnderHandover())
	t.False(hd.IsReady())

	uold := hd.OldNode()
	t.NotNil(uold)
	t.True(old.ConnInfo().Equal(uold.ConnInfo()))
}

func (t *testHandover) TestInvestigate() {
	t.Run("not in suffrage", func() {
		hd := t.newHandover(t.local, t.Suffrage(t.remote))
		t.NoError(hd.Start())
		defer func() {
			_ = hd.Stop()
		}()

		t.False(hd.UnderHandover())
	})

	t.Run("something wrong", func() {
		hd := t.newHandover(t.local, t.Suffrage(t.local))
		hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
			return nil, network.NodeInfoV0{}, errors.Errorf("something wrong")
		}
		t.NoError(hd.Start())
		defer func() {
			_ = hd.Stop()
		}()

		<-time.After(time.Second)
		t.False(hd.IsStarted())
	})

	t.Run("old node not found", func() {
		hd := t.newHandover(t.local, t.Suffrage(t.local))
		hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
			return nil, network.NodeInfoV0{}, util.IgnoreError.Errorf("no old node")
		}
		t.NoError(hd.Start())
		defer func() {
			_ = hd.Stop()
		}()
		t.False(hd.UnderHandover())

		<-time.After(time.Second)
		t.False(hd.IsStarted())
	})

	t.Run("old node found", func() {
		hd := t.newHandover(t.local, t.Suffrage(t.local))

		old := t.newChannel("https://old")
		hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
			return old, network.NodeInfoV0{}, nil
		}
		t.NoError(hd.Start())
		defer func() {
			t.NoError(hd.Stop())
		}()
		t.True(hd.UnderHandover())
		t.False(hd.IsReady())
		t.NotNil(hd.OldNode())
	})
}

func (t *testHandover) TestKeepVerifying() {
	t.Run("nil node info", func() {
		hd := t.newHandover(t.local, t.Suffrage(t.local))
		hd.intervalKeepVerifyDuplicatedNode = time.Millisecond * 100
		hd.maxFailedCountKeepVerifyDuplicatedNode = 1

		old := t.remote.Channel()
		old.(*channetwork.Channel).SetNodeInfoHandler(func() (network.NodeInfo, error) {
			return nil, nil
		})

		hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
			return old, network.NodeInfoV0{}, nil
		}

		t.NoError(hd.Start())
		defer func() {
			_ = hd.Stop()
		}()
		t.True(hd.UnderHandover())

		<-time.After(hd.intervalKeepVerifyDuplicatedNode + time.Second)
		t.False(hd.UnderHandover())
		t.False(hd.IsStarted())
	})

	t.Run("not alive after under handover", func() {
		suffrage := t.Suffrage(t.local)
		hd := t.newHandover(t.local, suffrage)
		hd.intervalKeepVerifyDuplicatedNode = time.Millisecond * 100
		hd.maxFailedCountKeepVerifyDuplicatedNode = 1

		calledch := make(chan struct{}, 1)
		var calledOnce sync.Once

		var nilnodeinfo int64

		old := t.remote.Channel()
		old.(*channetwork.Channel).SetNodeInfoHandler(func() (network.NodeInfo, error) {
			if atomic.LoadInt64(&nilnodeinfo) > 0 {
				return nil, nil
			}

			ni := network.NewNodeInfoV0(
				t.local.Node(),
				t.local.Policy().NetworkID(),
				base.StateJoining,
				nil,
				util.Version("0.1.1"),
				map[string]interface{}{"showme": 1},
				nil,
				suffrage,
				old.ConnInfo(),
			)

			calledOnce.Do(func() {
				calledch <- struct{}{}
			})

			return ni, nil
		})

		hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
			return old, network.NodeInfoV0{}, nil
		}

		t.NoError(hd.Start())
		defer func() {
			_ = hd.Stop()
		}()
		t.True(hd.UnderHandover())

		<-calledch
		<-time.After(time.Second * 1)
		t.True(hd.UnderHandover()) // NOTE still alive

		atomic.AddInt64(&nilnodeinfo, 1) // NOTE set not alive

		<-time.After(hd.intervalKeepVerifyDuplicatedNode + time.Second)
		t.False(hd.UnderHandover())
		t.False(hd.IsStarted())
	})
}

func (t *testHandover) TestUpdatingNodes() {
	suffrage := t.Suffrage(t.local, t.remote)
	lci, _ := network.NewHTTPConnInfoFromString("https://local", true)
	rci, _ := network.NewHTTPConnInfoFromString("https://remote", true)

	ls := t.Locals(1)
	old := ls[0].Channel()

	ni := network.NewNodeInfoV0(
		t.local.Node(),
		t.local.Policy().NetworkID(),
		base.StateJoining,
		nil,
		util.Version("0.1.1"),
		map[string]interface{}{"showme": 1},
		[]network.RemoteNode{
			network.NewRemoteNode(t.local.Node(), lci),
			network.NewRemoteNode(t.remote.Node(), rci),
		},
		suffrage,
		old.ConnInfo(),
	)

	old.(*channetwork.Channel).SetNodeInfoHandler(func() (network.NodeInfo, error) {
		return ni, nil
	})

	t.Run("update nodes", func() {
		hd := t.newHandover(t.local, suffrage)
		hd.intervalKeepVerifyDuplicatedNode = time.Millisecond * 100
		hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
			return old, ni, nil
		}

		added, err := hd.addRemoteChannels([]network.Channel{old})
		t.NoError(err)
		t.True(added)

		// NOTE remove channel
		_ = t.local.Nodes().SetChannel(t.remote.Node().Address(), nil)
		rch, found := t.local.Nodes().Channel(t.remote.Node().Address())
		t.True(found)
		t.Nil(rch)

		t.NoError(hd.Start())
		defer func() {
			t.NoError(hd.Stop())
		}()
		t.True(hd.UnderHandover())

		<-time.After(time.Second * 1)
		t.True(hd.UnderHandover())

		rch, found = t.local.Nodes().Channel(t.remote.Node().Address())
		t.True(found)
		t.NotNil(rch) // NOTE updated by nodeinfo
	})
}

func (t *testHandover) TestPing() {
	suffrage := t.Suffrage(t.local, t.remote)

	ls := t.Locals(1)
	old := ls[0].Channel()

	ni := network.NewNodeInfoV0(
		t.local.Node(),
		t.local.Policy().NetworkID(),
		base.StateJoining,
		nil,
		util.Version("0.1.1"),
		map[string]interface{}{"showme": 1},
		nil,
		suffrage,
		old.ConnInfo(),
	)

	old.(*channetwork.Channel).SetNodeInfoHandler(func() (network.NodeInfo, error) {
		return ni, nil
	})

	var called int64
	old.(*channetwork.Channel).SetPingHandover(func(network.PingHandoverSeal) (bool, error) {
		atomic.AddInt64(&called, 1)

		return true, nil
	})

	t.Run("called", func() {
		hd := t.newHandover(t.local, suffrage)
		hd.intervalPingHandover = time.Millisecond * 100
		hd.checkDuplicatedNodeFunc = func() (network.Channel, network.NodeInfo, error) {
			return old, ni, nil
		}

		t.NoError(hd.Start())
		defer func() {
			t.NoError(hd.Stop())
		}()
		t.True(hd.UnderHandover())

		<-time.After(time.Second * 1)
		t.True(atomic.LoadInt64(&called) > 2)
	})
}

func TestHandover(t *testing.T) {
	suite.Run(t, new(testHandover))
}
