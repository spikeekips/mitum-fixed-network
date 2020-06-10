package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
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

	manifest := t.lastManifest(t.localstate.Storage())

	t.True(t.localstate.Node().Publickey().Equal(ballot.Signer()))
	t.Equal(base.StageINIT, ballot.Stage())
	t.Equal(manifest.Height()+1, ballot.Height())
	t.Equal(base.Round(0), ballot.Round())
	t.True(t.localstate.Node().Address().Equal(ballot.Node()))

	t.True(manifest.Hash().Equal(ballot.PreviousBlock()))
}

// INIT Ballot, which,
// - ballot.Height() == local.Height() + 1
// - has ACCEPT vp(Voteproof)
// - vp.Result == VoteResultMajority
//
// StateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testStateJoiningHandler) TestINITBallotWithACCEPTVoteproofExpectedHeight() {
	r := base.ThresholdRatio(67)
	_ = t.localstate.Policy().SetThresholdRatio(r)
	_ = t.remoteState.Policy().SetThresholdRatio(r)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	manifest := t.lastManifest(t.localstate.Storage())

	// ACCEPT Voteproof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		manifest.Height(),
		manifest.Round(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.localstate.Node().Address(),
		manifest.Height()+1,
		cs.currentRound(),
		manifest.Hash(),
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
	r := base.ThresholdRatio(67)
	_ = t.localstate.Policy().SetThresholdRatio(r)
	_ = t.remoteState.Policy().SetThresholdRatio(r)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	manifest := t.lastManifest(t.remoteState.Storage())

	// ACCEPT Voteproof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		manifest.Height()-1,
		base.Round(0),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.remoteState.Node().Address(),
		manifest.Height()-1,
		cs.currentRound(),
		manifest.Hash(),
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
	r := base.ThresholdRatio(67)
	_ = t.localstate.Policy().SetThresholdRatio(r)
	_ = t.remoteState.Policy().SetThresholdRatio(r)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	manifest := t.lastManifest(t.localstate.Storage())

	// ACCEPT Voteproof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		manifest.Height()+1,
		base.Round(0),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.remoteState.Node().Address(),
		manifest.Height()+2,
		cs.currentRound(),
		valuehash.RandomSHA256(),
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
	r := base.ThresholdRatio(67)
	_ = t.localstate.Policy().SetThresholdRatio(r)
	_ = t.remoteState.Policy().SetThresholdRatio(r)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	cs.setCurrentRound(base.Round(1))
	manifest := t.lastManifest(t.remoteState.Storage())

	initFact := ballot.NewINITBallotV0(
		nil,
		manifest.Height()+1,
		cs.currentRound()-1,
		manifest.Hash(),
		nil,
	).Fact().(ballot.INITBallotFactV0)

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.remoteState.Node().Address(),
		initFact.Height(),
		initFact.Round()+1,
		manifest.Hash(),
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
	r := base.ThresholdRatio(67)
	_ = t.localstate.Policy().SetThresholdRatio(r)
	_ = t.remoteState.Policy().SetThresholdRatio(r)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	cs.setCurrentRound(base.Round(1))
	manifest := t.lastManifest(t.remoteState.Storage())

	initFact := ballot.NewINITBallotV0(
		nil,
		manifest.Height()-2,
		base.Round(0),
		manifest.Hash(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.remoteState.Node().Address(),
		manifest.Height()-1,
		cs.currentRound(),
		manifest.Hash(),
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
	r := base.ThresholdRatio(67)
	_ = t.localstate.Policy().SetThresholdRatio(r)
	_ = t.remoteState.Policy().SetThresholdRatio(r)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	cs.setCurrentRound(base.Round(1))

	manifest := t.lastManifest(t.localstate.Storage())

	initFact := ballot.NewINITBallotV0(
		nil,
		manifest.Height()+3,
		base.Round(0),
		manifest.Hash(),
		nil,
	).Fact()

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib := ballot.NewINITBallotV0(
		t.remoteState.Node().Address(),
		manifest.Height()+3,
		cs.currentRound(),
		manifest.Hash(),
		vp,
	)
	t.NoError(ib.Sign(t.remoteState.Node().Privatekey(), t.remoteState.Policy().NetworkID()))

	t.NoError(cs.NewSeal(ib))
}

// With new INIT Voteproof
// - vp.Height() == local + 1
// StateJoiningHandler will moves to consensus state.
func (t *testStateJoiningHandler) TestINITVoteproofExpected() {
	r := base.ThresholdRatio(67)
	_ = t.localstate.Policy().SetThresholdRatio(r)
	_ = t.remoteState.Policy().SetThresholdRatio(r)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	manifest := t.lastManifest(t.localstate.Storage())
	initFact := ballot.NewINITBallotV0(
		nil,
		manifest.Height()+1,
		base.Round(2), // round is not important to go
		manifest.Hash(),
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
	r := base.ThresholdRatio(67)
	_ = t.localstate.Policy().SetThresholdRatio(r)
	_ = t.remoteState.Policy().SetThresholdRatio(r)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	manifest := t.lastManifest(t.localstate.Storage())
	initFact := ballot.NewINITBallotV0(
		nil,
		manifest.Height(),
		base.Round(2), // round is not important to go
		manifest.Hash(),
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
	r := base.ThresholdRatio(67)
	_ = t.localstate.Policy().SetThresholdRatio(r)
	_ = t.remoteState.Policy().SetThresholdRatio(r)

	manifest := t.lastManifest(t.localstate.Storage())
	returnedBlock, err := block.NewTestBlockV0(
		manifest.Height()+1,
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
	r := base.ThresholdRatio(67)
	_ = t.localstate.Policy().SetThresholdRatio(r)
	_ = t.remoteState.Policy().SetThresholdRatio(r)

	cs, err := NewStateJoiningHandler(t.localstate, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(StateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(StateChangeContext{})
	}()

	manifest := t.lastManifest(t.localstate.Storage())
	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		manifest.Height(),
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
