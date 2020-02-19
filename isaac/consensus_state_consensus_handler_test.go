package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type testConsensusStateConsensusHandler struct {
	baseTestConsensusStateHandler
}

func (t *testConsensusStateConsensusHandler) TestNew() {
	t.localState.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	suffrage := t.suffrage(t.remoteState, t.localState)

	proposalMaker := NewProposalMaker(t.localState)
	cs, err := NewConsensusStateConsensusHandler(
		t.localState, DummyProposalProcessor{}, suffrage, t.sealStorage, proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	ib, err := NewINITBallotV0FromLocalState(t.localState, Round(0), nil)
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	vp, err := t.newVoteProof(StageINIT, initFact, t.localState, t.remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(ConsensusStateChangeContext{
		fromState: ConsensusStateJoining,
		toState:   ConsensusStateJoining,
		voteProof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	lb := cs.localState.LastINITVoteProof()

	t.Equal(vp.Height(), lb.Height())
	t.Equal(vp.Round(), lb.Round())
	t.Equal(vp.Stage(), lb.Stage())
	t.Equal(vp.Result(), lb.Result())
	t.Equal(vp.Majority(), lb.Majority())

	<-time.After(time.Millisecond * 100)
}

func (t *testConsensusStateConsensusHandler) TestWaitingProposalButTimeedOut() {
	t.localState.Policy().SetTimeoutWaitingProposal(time.Millisecond * 3)
	t.localState.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 5)

	suffrage := t.suffrage(t.remoteState, t.localState)

	proposalMaker := NewProposalMaker(t.localState)
	cs, err := NewConsensusStateConsensusHandler(t.localState, DummyProposalProcessor{}, suffrage, t.sealStorage, proposalMaker)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	ib, err := NewINITBallotV0FromLocalState(t.localState, Round(0), nil)
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	vp, err := t.newVoteProof(StageINIT, initFact, t.localState, t.remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(ConsensusStateChangeContext{
		fromState: ConsensusStateJoining,
		toState:   ConsensusStateConsensus,
		voteProof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	<-time.After(time.Millisecond * 10)

	r := <-sealChan
	t.NotNil(r)

	rb := r.(INITBallotV0)

	t.Equal(StageINIT, rb.Stage())
	t.Equal(vp.Height(), rb.Height())
	t.Equal(vp.Round()+1, rb.Round()) // means that handler moves to next round
}

// with Proposal, ACCEPTBallot will be broadcasted with newly processed
// Proposal.
func (t *testConsensusStateConsensusHandler) TestWithProposalWaitACCEPTBallot() {
	t.localState.Policy().SetWaitBroadcastingACCEPTBallot(time.Millisecond * 1)

	ib, err := NewINITBallotV0FromLocalState(t.localState, Round(0), nil)
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	proposalMaker := NewProposalMaker(t.localState)
	cs, err := NewConsensusStateConsensusHandler(
		t.localState,
		DummyProposalProcessor{},
		t.suffrage(t.remoteState, t.remoteState), // localnode is not in ActingSuffrage.
		t.sealStorage,
		proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	vp, err := t.newVoteProof(StageINIT, initFact, t.localState, t.remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(ConsensusStateChangeContext{
		fromState: ConsensusStateJoining,
		toState:   ConsensusStateConsensus,
		voteProof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	pr, err := NewProposalFromLocalState(t.remoteState, initFact.round, nil, nil)
	t.NoError(err)

	returnedBlock, err := NewTestBlockV0(initFact.Height(), initFact.Round(), pr.Hash(), valuehash.RandomSHA256())
	t.NoError(err)
	cs.proposalProcessor = NewDummyProposalProcessor(returnedBlock, nil)

	t.NoError(cs.NewSeal(pr))

	r := <-sealChan
	t.NotNil(r)

	rb := r.(ACCEPTBallotV0)
	t.Equal(StageACCEPT, rb.Stage())

	t.Equal(pr.Height(), rb.Height())
	t.Equal(pr.Round(), rb.Round())
	t.True(pr.Hash().Equal(rb.Proposal()))
	t.True(returnedBlock.Hash().Equal(rb.NewBlock()))
}

// with Proposal, ACCEPTBallot will be broadcasted with newly processed
// Proposal.
func (t *testConsensusStateConsensusHandler) TestWithProposalWaitSIGNBallot() {
	ib, err := NewINITBallotV0FromLocalState(t.localState, Round(0), nil)
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	proposalMaker := NewProposalMaker(t.localState)
	cs, err := NewConsensusStateConsensusHandler(
		t.localState,
		DummyProposalProcessor{},
		t.suffrage(t.remoteState, t.localState, t.remoteState), // localnode is not in ActingSuffrage.
		t.sealStorage,
		proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	vp, err := t.newVoteProof(StageINIT, initFact, t.localState, t.remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(ConsensusStateChangeContext{
		fromState: ConsensusStateJoining,
		toState:   ConsensusStateConsensus,
		voteProof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	pr, err := NewProposalFromLocalState(t.remoteState, initFact.round, nil, nil)
	t.NoError(err)

	returnedBlock, err := NewTestBlockV0(initFact.Height(), initFact.Round(), pr.Hash(), valuehash.RandomSHA256())
	t.NoError(err)
	cs.proposalProcessor = NewDummyProposalProcessor(returnedBlock, nil)

	t.NoError(cs.NewSeal(pr))

	r := <-sealChan
	t.NotNil(r)

	rb := r.(SIGNBallotV0)
	t.Equal(StageSIGN, rb.Stage())

	t.Equal(pr.Height(), rb.Height())
	t.Equal(pr.Round(), rb.Round())
	t.True(pr.Hash().Equal(rb.Proposal()))
	t.True(returnedBlock.Hash().Equal(rb.NewBlock()))
}

func (t *testConsensusStateConsensusHandler) TestDraw() {
	proposalMaker := NewProposalMaker(t.localState)
	cs, err := NewConsensusStateConsensusHandler(
		t.localState,
		DummyProposalProcessor{},
		t.suffrage(t.remoteState, t.localState, t.remoteState), // localnode is not in ActingSuffrage.
		t.sealStorage,
		proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	var vp VoteProof
	{
		ib, err := NewINITBallotV0FromLocalState(t.localState, Round(0), nil)
		t.NoError(err)
		fact := ib.INITBallotFactV0

		vp, _ = t.newVoteProof(StageINIT, fact, t.localState, t.remoteState)
	}

	t.NoError(cs.Activate(ConsensusStateChangeContext{
		fromState: ConsensusStateJoining,
		toState:   ConsensusStateConsensus,
		voteProof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	var drew VoteProofV0
	{
		dummyBlock, _ := NewTestBlockV0(vp.Height(), vp.Round(), valuehash.RandomSHA256(), valuehash.RandomSHA256())

		ab, err := NewACCEPTBallotV0FromLocalState(t.localState, vp.Round(), dummyBlock, nil)
		t.NoError(err)
		fact := ab.ACCEPTBallotFactV0

		drew, _ = t.newVoteProof(StageINIT, fact, t.localState, t.remoteState)
		drew.result = VoteProofDraw
	}

	t.NoError(cs.NewVoteProof(drew))

	r := <-sealChan
	t.NotNil(r)
	t.Implements((*INITBallot)(nil), r)

	ib := r.(INITBallotV0)
	t.Equal(StageINIT, ib.Stage())
	t.Equal(vp.Height(), ib.Height())
	t.Equal(vp.Round()+1, ib.Round())
}

func TestConsensusStateConsensusHandler(t *testing.T) {
	suite.Run(t, new(testConsensusStateConsensusHandler))
}
