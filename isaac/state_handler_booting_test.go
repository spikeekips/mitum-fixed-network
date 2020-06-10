package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/network"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/util"
)

type testStateBootingHandler struct {
	baseTestStateHandler
}

func (t *testStateBootingHandler) TestWithBlock() {
	cs, err := NewStateBootingHandler(t.localstate, t.suffrage(t.localstate))
	t.NoError(err)

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	doneChan := make(chan struct{})
	go func() {
	end:
		for {
			select {
			case <-time.After(time.Second):
				break end
			case ctx := <-stateChan:
				if ctx.To() == base.StateJoining {
					doneChan <- struct{}{}

					break end
				}
			}
		}
	}()

	t.NoError(cs.Activate(NewStateChangeContext(base.StateStopped, base.StateBooting, nil, nil)))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	select {
	case <-time.After(time.Second):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case <-doneChan:
		break
	}
}

func (t *testStateBootingHandler) TestWithoutBlock() {
	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(0), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	policy := DefaultPolicy().
		SetThresholdRatio(DefaultPolicy().ThresholdRatio() - 1).
		SetNumberOfActingSuffrageNodes(DefaultPolicy().NumberOfActingSuffrageNodes() + 1)

	ni := network.NewNodeInfoV0(
		base.RandomNode("n0"),
		TestNetworkID,
		base.StateBooting,
		blk.Manifest(),
		util.Version("0.1.1"),
		"quic://local",
		policy,
	)

	nch := t.remoteState.Node().Channel().(*channetwork.NetworkChanChannel)
	nch.SetNodeInfoHandler(func() (network.NodeInfo, error) {
		return ni, nil
	})

	cs, err := NewStateBootingHandler(t.localstate, t.suffrage(t.localstate, t.remoteState))
	t.NoError(err)
	t.NoError(t.localstate.Storage().Clean())

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	doneChan := make(chan struct{})
	go func() {
	end:
		for {
			select {
			case <-time.After(time.Second):
				break end
			case ctx := <-stateChan:
				if ctx.To() == base.StateSyncing {
					doneChan <- struct{}{}

					break end
				}
			}
		}
	}()

	t.NoError(cs.Activate(NewStateChangeContext(base.StateStopped, base.StateBooting, nil, nil)))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	select {
	case <-time.After(time.Second):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case <-doneChan:
		break
	}

	t.Equal(policy.ThresholdRatio(), t.localstate.Policy().ThresholdRatio())
	t.Equal(policy.NumberOfActingSuffrageNodes(), t.localstate.Policy().NumberOfActingSuffrageNodes())
}

func TestStateBootingHandler(t *testing.T) {
	suite.Run(t, new(testStateBootingHandler))
}
