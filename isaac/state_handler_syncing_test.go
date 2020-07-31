package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
)

type testStateSyncingHandler struct {
	baseTestStateHandler

	local  *Localstate
	remote *Localstate
}

func (t *testStateSyncingHandler) SetupTest() {
	t.baseTestStateHandler.SetupTest()

	ls := t.localstates(2)
	t.local, t.remote = ls[0], ls[1]
}

func (t *testStateSyncingHandler) TestINITMovesToConsensus() {
	t.local.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	cs := NewStateSyncingHandler(t.local)
	t.NotNil(cs)

	stateChan := make(chan *StateChangeContext)
	cs.SetStateChan(stateChan)

	doneChan := make(chan struct{})
	go func() {
		for ctx := range stateChan {
			if ctx.To() == base.StateConsensus {
				doneChan <- struct{}{}

				break
			}
		}
	}()

	var voteproof base.Voteproof
	{
		b := t.newINITBallot(t.remote, base.Round(0), t.lastINITVoteproof(t.remote))

		vp, err := t.newVoteproof(b.Stage(), b.INITBallotFactV0, t.remote)
		t.NoError(err)

		voteproof = vp
	}

	t.NoError(cs.Activate(NewStateChangeContext(base.StateJoining, base.StateSyncing, voteproof, nil)))
	defer func() {
		_ = cs.Deactivate(nil)
	}()

	select {
	case <-time.After(time.Second):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case <-doneChan:
		break
	}
}

func (t *testStateSyncingHandler) TestWaitMovesToJoining() {
	t.local.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	cs := NewStateSyncingHandler(t.local)
	cs.waitVoteproofTimeout = time.Millisecond * 10
	t.NotNil(cs)

	stateChan := make(chan *StateChangeContext)
	cs.SetStateChan(stateChan)

	doneChan := make(chan struct{})
	go func() {
		for ctx := range stateChan {
			if ctx.To() == base.StateJoining {
				doneChan <- struct{}{}

				break
			}
		}
	}()

	t.NoError(cs.Activate(NewStateChangeContext(base.StateBooting, base.StateSyncing, nil, nil)))
	defer func() {
		_ = cs.Deactivate(nil)
	}()

	cs.whenFinished(base.NilHeight)

	select {
	case <-time.After(time.Second * 2):
		t.NoError(xerrors.Errorf("timeout to wait to be finished"))
	case <-doneChan:
		break
	}
}

func TestStateSyncingHandler(t *testing.T) {
	suite.Run(t, new(testStateSyncingHandler))
}
