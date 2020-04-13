package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
)

type testStateJoiningHandler struct {
	baseTestStateHandler
}

func (t *testStateJoiningHandler) TestNew() {
	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))

	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()
}

func (t *testStateJoiningHandler) TestKeepBroadcastingINITBallot() {
	_, _ = t.localstate.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 30)
	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	time.Sleep(time.Millisecond * 50)

	received := <-sealChan
	t.NotNil(received)

	t.Implements((*seal.Seal)(nil), received)
	t.IsType(ballot.INITBallotV0{}, received)

	ballot := received.(ballot.INITBallotV0)

	t.NoError(ballot.IsValid(t.localstate.Policy().NetworkID()))

	t.True(t.localstate.Node().Publickey().Equal(ballot.Signer()))
	t.Equal(base.StageINIT, ballot.Stage())
	t.Equal(t.localstate.LastBlock().Height()+1, ballot.Height())
	t.Equal(base.Round(0), ballot.Round())
	t.True(t.localstate.Node().Address().Equal(ballot.Node()))

	t.True(t.localstate.LastBlock().Hash().Equal(ballot.PreviousBlock()))
	t.Equal(t.localstate.LastBlock().Round(), ballot.PreviousRound())
}

// INIT Ballot, which,
// - ballot.Height() == local.Height() + 1
// - has ACCEPT vp(Voteproof)
// - vp.Result == VoteResultMajority
//
// StateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testStateJoiningHandler) TestINITBallotWithACCEPTVoteproofExpectedHeight() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	lastBlock := t.localstate.LastBlock()

	// ACCEPT Voteproof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		lastBlock.Height(),
		lastBlock.Round(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.localstate.Node().Address(),
		lastBlock.Height()+1,
		cs.currentRound(),
		lastBlock.Hash(),
		lastBlock.Round(),
		vp,
	)
	t.NoError(ib.Sign(t.remoteState.Node().Privatekey(), t.remoteState.Policy().NetworkID()))

	t.NoError(cs.NewSeal(ib))
}

// INIT Ballot, which,
// - ballot.Height() < local.Height() + 1
// - has ACCEPT vp(Voteproof)
// - vp.Result == VoteResultMajority
//
// StateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testStateJoiningHandler) TestINITBallotWithACCEPTVoteproofLowerHeight() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	lastBlock := t.remoteState.LastBlock()

	// ACCEPT Voteproof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		lastBlock.Height()-1,
		base.Round(0),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.remoteState.Node().Address(),
		lastBlock.Height()-1,
		cs.currentRound(),
		lastBlock.Hash(),
		lastBlock.Round(),
		vp,
	)
	t.NoError(ib.Sign(t.remoteState.Node().Privatekey(), t.remoteState.Policy().NetworkID()))

	t.NoError(cs.NewSeal(ib))
}

// INIT Ballot, which,
// - ballot.Height() > local.Height() + 1
// - has ACCEPT vp(Voteproof)
// - vp.Result == VoteResultMajority
//
// StateJoiningHandler will stop broadcasting it's INIT Ballot and
// moves to syncing.
func (t *testStateJoiningHandler) TestINITBallotWithACCEPTVoteproofHigherHeight() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	lastBlock := t.remoteState.LastBlock()

	// ACCEPT Voteproof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		lastBlock.Height()+1,
		base.Round(0),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.remoteState.Node().Address(),
		lastBlock.Height()+2,
		cs.currentRound(),
		valuehash.RandomSHA256(),
		base.Round(0),
		vp,
	)
	t.NoError(ib.Sign(t.remoteState.Node().Privatekey(), t.remoteState.Policy().NetworkID()))

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewSeal(ib))

	var ctx StateChangeContext
	select {
	case ctx = <-stateChan:
	case <-time.After(time.Millisecond * 100):
		t.NoError(xerrors.Errorf("failed to change state to syncing"))
	}

	t.Equal(base.StateJoining, ctx.fromState)
	t.Equal(base.StateSyncing, ctx.toState)
	t.Equal(base.StageACCEPT, ctx.voteproof.Stage())
	t.Equal(acceptFact, ctx.voteproof.Majority())
}

// INIT Ballot, which,
// - ballot.Height() == local.Height() + 1
// - has INIT vp(Voteproof)
// - ballot.Round == vp.Round + 1
// - vp.Result == VoteResultDraw || vp.Result == VoteResultMajority
//
// StateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testStateJoiningHandler) TestINITBallotWithINITVoteproofExpectedHeight() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	cs.setCurrentRound(base.Round(1))
	lastBlock := t.remoteState.LastBlock()

	initFact := ballot.NewINITBallotV0(
		nil,
		lastBlock.Height()+1,
		cs.currentRound()-1,
		lastBlock.Hash(),
		lastBlock.Round(),
		nil,
	).Fact().(ballot.INITBallotFactV0)

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.remoteState.Node().Address(),
		initFact.Height(),
		initFact.Round()+1,
		lastBlock.Hash(),
		lastBlock.Round(),
		vp,
	)
	t.NoError(ib.Sign(t.remoteState.Node().Privatekey(), t.remoteState.Policy().NetworkID()))

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewSeal(ib))
}

// INIT Ballot, which,
// - ballot.Height() < local.Height() + 1
// - has INIT vp(Voteproof)
// - vp.Result == VoteResultDraw || vp.Result == VoteResultMajority
//
// StateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testStateJoiningHandler) TestINITBallotWithINITVoteproofLowerHeight() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	cs.setCurrentRound(base.Round(1))
	lastBlock := t.remoteState.LastBlock()

	initFact := ballot.NewINITBallotV0(
		nil,
		lastBlock.Height()-2,
		base.Round(0),
		lastBlock.Hash(),
		lastBlock.Round(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.remoteState.Node().Address(),
		lastBlock.Height()-1,
		cs.currentRound(),
		lastBlock.Hash(),
		lastBlock.Round(),
		vp,
	)
	t.NoError(ib.Sign(t.remoteState.Node().Privatekey(), t.remoteState.Policy().NetworkID()))

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewSeal(ib))
}

// INIT Ballot, which,
// - ballot.Height() > local.Height() + 1
// - has INIT vp(Voteproof)
// - vp.Result == VoteResultDraw || vp.Result == VoteResultMajority
//
// StateJoiningHandler will stop broadcasting it's INIT Ballot and
// moves to syncing.
func (t *testStateJoiningHandler) TestINITBallotWithINITVoteproofHigherHeight() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	cs.setCurrentRound(base.Round(1))

	lastBlock := t.localstate.LastBlock()

	initFact := ballot.NewINITBallotV0(
		nil,
		lastBlock.Height()+3,
		base.Round(0),
		lastBlock.Hash(),
		lastBlock.Round(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.remoteState.Node().Address(),
		lastBlock.Height()+3,
		cs.currentRound(),
		lastBlock.Hash(),
		lastBlock.Round(),
		vp,
	)
	t.NoError(ib.Sign(t.remoteState.Node().Privatekey(), t.remoteState.Policy().NetworkID()))

	t.NoError(cs.NewSeal(ib))
}

// With new INIT Voteproof
// - vp.Height() == local + 1
// StateJoiningHandler will moves to consensus state.
func (t *testStateJoiningHandler) TestINITVoteproofExpected() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	initFact := ballot.NewINITBallotV0(
		nil,
		t.localstate.LastBlock().Height()+1,
		base.Round(2), // round is not important to go
		t.localstate.LastBlock().Hash(),
		t.localstate.LastBlock().Round(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteproof(vp))

	var ctx StateChangeContext
	select {
	case ctx = <-stateChan:
	case <-time.After(time.Millisecond * 100):
		t.NoError(xerrors.Errorf("failed to change state to syncing"))
	}

	t.Equal(base.StateJoining, ctx.fromState)
	t.Equal(base.StateConsensus, ctx.toState)
	t.Equal(base.StageINIT, ctx.voteproof.Stage())
	t.Equal(initFact, ctx.voteproof.Majority())
}

// With new INIT Voteproof
// - vp.Height() < local + 1
// StateJoiningHandler will wait another Voteproof
func (t *testStateJoiningHandler) TestINITVoteproofLowerHeight() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	initFact := ballot.NewINITBallotV0(
		nil,
		t.localstate.LastBlock().Height(),
		base.Round(2), // round is not important to go
		t.localstate.LastBlock().Hash(),
		t.localstate.LastBlock().Round(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteproof(vp))
}

// With new ACCEPT Voteproof
// - vp.Height() == local + 1
// StateJoiningHandler will processing Proposal.
func (t *testStateJoiningHandler) TestACCEPTVoteproofExpected() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	returnedBlock, err := NewTestBlockV0(
		t.localstate.LastBlock().Height()+1,
		base.Round(2),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	)
	t.NoError(err)
	proposalProcessor := NewDummyProposalProcessor(returnedBlock, nil)

	cs, err := NewStateJoiningHandler(t.localstate, proposalProcessor)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		returnedBlock.Height(),
		returnedBlock.Round(), // round is not important to go
		returnedBlock.Proposal(),
		returnedBlock.Hash(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteproof(vp))
}

// With new ACCEPT Voteproof
// - vp.Height() < local + 1
// StateJoiningHandler will wait another Voteproof
func (t *testStateJoiningHandler) TestACCEPTVoteproofLowerHeight() {
	thr, _ := base.NewThreshold(2, 67)
	_ = t.localstate.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		t.localstate.LastBlock().Height(),
		base.Round(2), // round is not important to go
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteproof(vp))
}

func TestStateJoiningHandler(t *testing.T) {
	suite.Run(t, new(testStateJoiningHandler))
}
