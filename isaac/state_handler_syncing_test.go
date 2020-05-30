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

	cs, err := NewStateSyncingHandler(t.localstate, NewDummyProposalProcessor(nil, nil))
	t.NoError(err)
	t.NotNil(cs)

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
				if ctx.To() == base.StateConsensus {
					doneChan <- struct{}{}

					break end
				}
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

func TestStateSyncingHandler(t *testing.T) {
	suite.Run(t, new(testStateSyncingHandler))
}
