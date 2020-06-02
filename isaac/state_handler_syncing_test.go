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
}

func (t *testStateSyncingHandler) TestINITMovesToConsensus() {
	t.localstate.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	cs := NewStateSyncingHandler(t.localstate)
	t.NotNil(cs)

	stateChan := make(chan StateChangeContext)
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
		b := t.newINITBallot(t.remoteState, base.Round(0), t.lastINITVoteproof(t.remoteState))

		vp, err := t.newVoteproof(b.Stage(), b.INITBallotFactV0, t.remoteState)
		t.NoError(err)

		voteproof = vp
	}

	t.NoError(cs.Activate(NewStateChangeContext(base.StateJoining, base.StateSyncing, voteproof, nil)))
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

func (t *testStateSyncingHandler) TestWaitMovesToJoining() {
	t.localstate.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	cs := NewStateSyncingHandler(t.localstate)
	cs.waitVoteproofTimeout = time.Millisecond * 10
	t.NotNil(cs)

	stateChan := make(chan StateChangeContext)
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
		_ = cs.Deactivate(StateChangeContext{})
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
