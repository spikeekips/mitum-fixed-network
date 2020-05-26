package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/valuehash"
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
		b := t.newINITBallot(t.remoteState, base.Round(0))

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

func (t *testStateSyncingHandler) TestProcessProposal() {
	t.localstate.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	lastManifest := t.lastManifest(t.localstate.Storage())

	proposal := ballot.NewProposalV0(
		base.NewShortAddress("test-for-proposal"),
		lastManifest.Height()+1,
		base.Round(0),
		nil, nil,
	)
	t.NoError(proposal.Sign(t.remoteState.Node().Privatekey(), nil))

	returnedBlock, err := block.NewTestBlockV0(
		lastManifest.Height()+1,
		base.Round(0), proposal.Hash(), valuehash.RandomSHA256())
	t.NoError(err)

	dp := NewDummyProposalProcessor(returnedBlock, nil)
	cs, err := NewStateSyncingHandler(t.localstate, dp)
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

	{ // without init voteproof
		t.NoError(cs.NewSeal(proposal))
		t.False(dp.IsProcessed(proposal.Hash()))
	}

	{
		b := t.newINITBallot(t.remoteState, base.Round(0))

		vp, err := t.newVoteproof(b.Stage(), b.INITBallotFactV0, t.remoteState)
		t.NoError(err)

		t.localstate.SetLastINITVoteproof(vp)
	}

	t.NoError(cs.NewSeal(proposal))
	t.True(dp.IsProcessed(proposal.Hash()))

	// and then, with the expected ACCEPT Voteproof, the block will be saved.}
	var acceptVoteproof base.Voteproof
	{
		ab := t.newACCEPTBallot(t.remoteState, base.Round(0), returnedBlock.Proposal(), returnedBlock.Hash())
		vp, err := t.newVoteproof(ab.Stage(), ab.ACCEPTBallotFactV0, t.remoteState)
		t.NoError(err)

		acceptVoteproof = vp
	}

	t.NoError(cs.NewVoteproof(acceptVoteproof))

	t.True(dp.BlockStorages(returnedBlock.Proposal()).Committed())
}

func TestStateSyncingHandler(t *testing.T) {
	suite.Run(t, new(testStateSyncingHandler))
}
