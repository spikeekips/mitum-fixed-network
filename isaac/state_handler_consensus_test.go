package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
)

type testStateConsensusHandler struct {
	baseTestStateHandler
}

func (t *testStateConsensusHandler) TestNew() {
	t.localstate.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	suffrage := t.suffrage(t.remoteState, t.localstate)

	proposalMaker := NewProposalMaker(t.localstate)
	cs, err := NewStateConsensusHandler(
		t.localstate, NewDummyProposalProcessor(nil, nil), suffrage, proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	ib, err := NewINITBallotV0FromLocalstate(t.localstate, base.Round(0))
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	_ = t.localstate.SetLastINITVoteproof(vp)

	t.NoError(cs.Activate(StateChangeContext{
		fromState: base.StateJoining,
		toState:   base.StateJoining,
		voteproof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	lb := t.localstate.LastINITVoteproof()

	t.Equal(vp.Height(), lb.Height())
	t.Equal(vp.Round(), lb.Round())
	t.Equal(vp.Stage(), lb.Stage())
	t.Equal(vp.Result(), lb.Result())
	t.Equal(vp.Majority(), lb.Majority())

	<-time.After(time.Millisecond * 100)
}

func (t *testStateConsensusHandler) TestWaitingProposalButTimedOut() {
	t.localstate.Policy().SetTimeoutWaitingProposal(time.Millisecond * 3)
	t.localstate.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 5)

	suffrage := t.suffrage(t.remoteState, t.localstate)

	proposalMaker := NewProposalMaker(t.localstate)
	cs, err := NewStateConsensusHandler(t.localstate, NewDummyProposalProcessor(nil, nil), suffrage, proposalMaker)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	ib, err := NewINITBallotV0FromLocalstate(t.localstate, base.Round(0))
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(StateChangeContext{
		fromState: base.StateJoining,
		toState:   base.StateConsensus,
		voteproof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	select {
	case <-time.After(time.Millisecond * 100):
		t.NoError(xerrors.Errorf("failed to get INITBallot for next round"))
	case r := <-sealChan:
		t.NotNil(r)

		rb := r.(INITBallotV0)

		t.Equal(base.StageINIT, rb.Stage())
		t.Equal(vp.Height(), rb.Height())
		t.Equal(vp.Round()+1, rb.Round()) // means that handler moves to next round
	}
}

// with Proposal, ACCEPTBallot will be broadcasted with newly processed
// Proposal.
func (t *testStateConsensusHandler) TestWithProposalWaitACCEPTBallot() {
	t.localstate.Policy().SetWaitBroadcastingACCEPTBallot(time.Millisecond * 1)

	ib, err := NewINITBallotV0FromLocalstate(t.localstate, base.Round(0))
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	proposalMaker := NewProposalMaker(t.localstate)
	cs, err := NewStateConsensusHandler(
		t.localstate,
		NewDummyProposalProcessor(nil, nil),
		t.suffrage(t.remoteState, t.remoteState), // localnode is not in ActingSuffrage.
		proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(StateChangeContext{
		fromState: base.StateJoining,
		toState:   base.StateConsensus,
		voteproof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	pr, err := NewProposalFromLocalstate(t.remoteState, initFact.round, nil, nil)
	t.NoError(err)

	returnedBlock, err := NewTestBlockV0(initFact.Height(), initFact.Round(), pr.Hash(), valuehash.RandomSHA256())
	t.NoError(err)
	cs.proposalProcessor = NewDummyProposalProcessor(returnedBlock, nil)

	t.NoError(cs.NewSeal(pr))

	r := <-sealChan
	t.NotNil(r)

	rb := r.(ACCEPTBallotV0)
	t.Equal(base.StageACCEPT, rb.Stage())

	t.Equal(pr.Height(), rb.Height())
	t.Equal(pr.Round(), rb.Round())
	t.True(pr.Hash().Equal(rb.Proposal()))
	t.True(returnedBlock.Hash().Equal(rb.NewBlock()))
}

// with Proposal, ACCEPTBallot will be broadcasted with newly processed
// Proposal.
func (t *testStateConsensusHandler) TestWithProposalWaitSIGNBallot() {
	ib, err := NewINITBallotV0FromLocalstate(t.localstate, base.Round(0))
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	proposalMaker := NewProposalMaker(t.localstate)
	cs, err := NewStateConsensusHandler(
		t.localstate,
		NewDummyProposalProcessor(nil, nil),
		t.suffrage(t.remoteState, t.localstate, t.remoteState), // localnode is not in ActingSuffrage.
		proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(StateChangeContext{
		fromState: base.StateJoining,
		toState:   base.StateConsensus,
		voteproof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	pr, err := NewProposalFromLocalstate(t.remoteState, initFact.round, nil, nil)
	t.NoError(err)

	returnedBlock, err := NewTestBlockV0(initFact.Height(), initFact.Round(), pr.Hash(), valuehash.RandomSHA256())
	t.NoError(err)
	cs.proposalProcessor = NewDummyProposalProcessor(returnedBlock, nil)

	t.NoError(cs.NewSeal(pr))

	r := <-sealChan
	t.NotNil(r)

	rb := r.(SIGNBallotV0)
	t.Equal(base.StageSIGN, rb.Stage())

	t.Equal(pr.Height(), rb.Height())
	t.Equal(pr.Round(), rb.Round())
	t.True(pr.Hash().Equal(rb.Proposal()))
	t.True(returnedBlock.Hash().Equal(rb.NewBlock()))
}

func (t *testStateConsensusHandler) TestDraw() {
	proposalMaker := NewProposalMaker(t.localstate)
	cs, err := NewStateConsensusHandler(
		t.localstate,
		NewDummyProposalProcessor(nil, nil),
		t.suffrage(t.remoteState, t.localstate, t.remoteState), // localnode is not in ActingSuffrage.
		proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	var vp base.Voteproof
	{
		ib, err := NewINITBallotV0FromLocalstate(t.localstate, base.Round(0))
		t.NoError(err)
		fact := ib.INITBallotFactV0

		vp, _ = t.newVoteproof(base.StageINIT, fact, t.localstate, t.remoteState)
	}

	t.NoError(cs.Activate(StateChangeContext{
		fromState: base.StateJoining,
		toState:   base.StateConsensus,
		voteproof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	var drew base.VoteproofV0
	{
		dummyBlock, _ := NewTestBlockV0(vp.Height(), vp.Round(), valuehash.RandomSHA256(), valuehash.RandomSHA256())

		ab, err := NewACCEPTBallotV0FromLocalstate(t.localstate, vp.Round(), dummyBlock)
		t.NoError(err)
		fact := ab.ACCEPTBallotFactV0

		drew, _ = t.newVoteproof(base.StageINIT, fact, t.localstate, t.remoteState)
		drew.SetResult(base.VoteResultDraw)
	}

	t.NoError(cs.NewVoteproof(drew))

	r := <-sealChan
	t.NotNil(r)
	t.Implements((*INITBallot)(nil), r)

	ib := r.(INITBallotV0)
	t.Equal(base.StageINIT, ib.Stage())
	t.Equal(vp.Height(), ib.Height())
	t.Equal(vp.Round()+1, ib.Round())
}

func TestStateConsensusHandler(t *testing.T) {
	suite.Run(t, new(testStateConsensusHandler))
}
