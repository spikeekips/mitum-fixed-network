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
	t.IsType(INITBallotV0{}, received)

	ballot := received.(INITBallotV0)

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
	ib, err := NewINITBallotV0FromLocalstate(t.localstate, cs.currentRound())
	t.NoError(err)

	// ACCEPT Voteproof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: lastBlock.Height(),
			round:  lastBlock.Round(),
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib.voteproof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

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

	ib, err := NewINITBallotV0FromLocalstate(t.remoteState, cs.currentRound())
	t.NoError(err)

	ib.INITBallotFactV0.height = t.remoteState.LastBlock().Height() - 1

	// ACCEPT Voteproof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height - 1,
			round:  base.Round(0),
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib.voteproof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

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

	ib, err := NewINITBallotV0FromLocalstate(t.remoteState, cs.currentRound())
	t.NoError(err)

	ib.INITBallotFactV0.height = t.remoteState.LastBlock().Height() + 2
	ib.INITBallotFactV0.previousBlock = valuehash.RandomSHA256()
	ib.INITBallotFactV0.previousRound = base.Round(0)

	// ACCEPT Voteproof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height - 1,
			round:  base.Round(0),
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib.voteproof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

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
	lastBlock := t.localstate.LastBlock()
	ib, err := NewINITBallotV0FromLocalstate(t.remoteState, cs.currentRound())
	t.NoError(err)

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height,
			round:  ib.INITBallotFactV0.round - 1,
		},
		previousBlock: lastBlock.Hash(),
		previousRound: lastBlock.Round(),
	}

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib.voteproof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

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
	lastBlock := t.localstate.LastBlock()
	ib, err := NewINITBallotV0FromLocalstate(t.remoteState, cs.currentRound())
	t.NoError(err)

	ib.INITBallotFactV0.height = t.remoteState.LastBlock().Height() - 1

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height - 1,
			round:  base.Round(0),
		},
		previousBlock: lastBlock.Hash(),
		previousRound: lastBlock.Round(),
	}

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib.voteproof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

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
	ib, err := NewINITBallotV0FromLocalstate(t.remoteState, cs.currentRound())
	t.NoError(err)

	ib.INITBallotFactV0.height = t.remoteState.LastBlock().Height() + 3

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height,
			round:  base.Round(0),
		},
		previousBlock: lastBlock.Hash(),
		previousRound: lastBlock.Round(),
	}

	vp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	t.NoError(err)

	ib.voteproof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

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

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: t.localstate.LastBlock().Height() + 1,
			round:  base.Round(2), // round is not important to go
		},
		previousBlock: t.localstate.LastBlock().Hash(),
		previousRound: t.localstate.LastBlock().Round(),
	}

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

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: t.localstate.LastBlock().Height(),
			round:  base.Round(2), // round is not important to go
		},
		previousBlock: t.localstate.LastBlock().Hash(),
		previousRound: t.localstate.LastBlock().Round(),
	}

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

	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: returnedBlock.Height(),
			round:  returnedBlock.Round(), // round is not important to go
		},
		proposal: returnedBlock.Proposal(),
		newBlock: returnedBlock.Hash(),
	}

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

	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: t.localstate.LastBlock().Height(),
			round:  base.Round(2), // round is not important to go
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)
	t.NoError(err)

	stateChan := make(chan StateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteproof(vp))
}

func TestStateJoiningHandler(t *testing.T) {
	suite.Run(t, new(testStateJoiningHandler))
}
