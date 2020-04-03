package isaac

import (
	"testing"
	"time"

	"github.com/spikeekips/mitum/valuehash"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
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
				if ctx.To() == StateConsensus {
					doneChan <- struct{}{}

					break end
				}
			}
		}
	}()

	var voteproof Voteproof
	{
		b := t.newINITBallot(t.remoteState, Round(0))

		vp, err := t.newVoteproof(b.Stage(), b.INITBallotFactV0, t.remoteState)
		t.NoError(err)

		voteproof = vp
	}

	t.NoError(cs.Activate(NewStateChangeContext(StateJoining, StateSyncing, voteproof, nil)))
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

	proposal := ProposalV0{
		BaseBallotV0: BaseBallotV0{
			node: NewShortAddress("test-for-proposal"),
		},
		ProposalFactV0: ProposalFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: t.localstate.LastBlock().Height() + 1,
				round:  Round(0),
			},
		},
	}
	t.NoError(proposal.Sign(t.remoteState.Node().Privatekey(), nil))

	returnedBlock, err := NewTestBlockV0(t.localstate.LastBlock().Height()+1, Round(0), proposal.Hash(), valuehash.RandomSHA256())
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
				if ctx.To() == StateConsensus {
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
		b := t.newINITBallot(t.remoteState, Round(0))

		vp, err := t.newVoteproof(b.Stage(), b.INITBallotFactV0, t.remoteState)
		t.NoError(err)

		t.localstate.SetLastINITVoteproof(vp)
	}

	t.NoError(cs.NewSeal(proposal))
	t.True(dp.IsProcessed(proposal.Hash()))

	// and the, with the expected ACCEPT Voteproof, the block will be saved.}
	var acceptVoteproof Voteproof
	{
		ab, err := NewACCEPTBallotV0FromLocalstate(t.remoteState, Round(0), returnedBlock)
		vp, err := t.newVoteproof(ab.Stage(), ab.ACCEPTBallotFactV0, t.remoteState)
		t.NoError(err)

		acceptVoteproof = vp
	}

	t.NoError(cs.NewVoteproof(acceptVoteproof))

	t.compareBlock(dp.returnBlock, t.localstate.LastBlock())
}

func TestStateSyncingHandler(t *testing.T) {
	suite.Run(t, new(testStateSyncingHandler))
}
