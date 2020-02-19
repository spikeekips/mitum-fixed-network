package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type testConsensusStateJoiningHandler struct {
	baseTestConsensusStateHandler
}

func (t *testConsensusStateJoiningHandler) TestNew() {
	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()
}

func (t *testConsensusStateJoiningHandler) TestKeepBroadcastingINITBallot() {
	_, _ = t.localState.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 30)
	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	time.Sleep(time.Millisecond * 50)

	received := <-sealChan
	t.NotNil(received)

	t.Implements((*seal.Seal)(nil), received)
	t.IsType(INITBallotV0{}, received)

	ballot := received.(INITBallotV0)

	t.NoError(ballot.IsValid(nil))

	t.True(t.localState.Node().Publickey().Equal(ballot.Signer()))
	t.Equal(StageINIT, ballot.Stage())
	t.Equal(t.localState.LastBlock().Height()+1, ballot.Height())
	t.Equal(Round(0), ballot.Round())
	t.True(t.localState.Node().Address().Equal(ballot.Node()))

	t.True(t.localState.LastBlock().Hash().Equal(ballot.PreviousBlock()))
	t.Equal(t.localState.LastBlock().Round(), ballot.PreviousRound())
}

// INIT Ballot, which,
// - ballot.Height() == local.Height() + 1
// - has ACCEPT vp(VoteProof)
// - vp.Result == VoteProofMajority
//
// ConsensusStateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testConsensusStateJoiningHandler) TestINITBallotWithACCEPTVoteProofExpectedHeight() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	lastBlock := t.localState.LastBlock()
	ib, err := NewINITBallotV0FromLocalState(t.localState, cs.currentRound(), nil)
	t.NoError(err)

	// ACCEPT VoteProof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: lastBlock.Height(),
			round:  lastBlock.Round(),
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, t.localState, t.remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

	t.NoError(cs.NewSeal(ib))
}

// INIT Ballot, which,
// - ballot.Height() < local.Height() + 1
// - has ACCEPT vp(VoteProof)
// - vp.Result == VoteProofMajority
//
// ConsensusStateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testConsensusStateJoiningHandler) TestINITBallotWithACCEPTVoteProofLowerHeight() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	ib, err := NewINITBallotV0FromLocalState(t.remoteState, cs.currentRound(), nil)
	t.NoError(err)

	ib.INITBallotFactV0.height = t.remoteState.LastBlock().Height() - 1

	// ACCEPT VoteProof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height - 1,
			round:  Round(0),
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, t.localState, t.remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

	t.NoError(cs.NewSeal(ib))
}

// INIT Ballot, which,
// - ballot.Height() > local.Height() + 1
// - has ACCEPT vp(VoteProof)
// - vp.Result == VoteProofMajority
//
// ConsensusStateJoiningHandler will stop broadcasting it's INIT Ballot and
// moves to syncing.
func (t *testConsensusStateJoiningHandler) TestINITBallotWithACCEPTVoteProofHigherHeight() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	ib, err := NewINITBallotV0FromLocalState(t.remoteState, cs.currentRound(), nil)
	t.NoError(err)

	ib.INITBallotFactV0.height = t.remoteState.LastBlock().Height() + 2
	ib.INITBallotFactV0.previousBlock = valuehash.RandomSHA256()
	ib.INITBallotFactV0.previousRound = Round(0)

	// ACCEPT VoteProof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height - 1,
			round:  Round(0),
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, t.localState, t.remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewSeal(ib))

	var ctx ConsensusStateChangeContext
	select {
	case ctx = <-stateChan:
	case <-time.After(time.Millisecond * 100):
		t.NoError(xerrors.Errorf("failed to change state to syncing"))
	}

	t.Equal(ConsensusStateJoining, ctx.fromState)
	t.Equal(ConsensusStateSyncing, ctx.toState)
	t.Equal(StageACCEPT, ctx.voteProof.Stage())
	t.Equal(acceptFact, ctx.voteProof.Majority())
}

// INIT Ballot, which,
// - ballot.Height() == local.Height() + 1
// - has INIT vp(VoteProof)
// - ballot.Round == vp.Round + 1
// - vp.Result == VoteProofDraw || vp.Result == VoteProofMajority
//
// ConsensusStateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testConsensusStateJoiningHandler) TestINITBallotWithINITVoteProofExpectedHeight() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	cs.setCurrentRound(Round(1))
	lastBlock := t.localState.LastBlock()
	ib, err := NewINITBallotV0FromLocalState(t.remoteState, cs.currentRound(), nil)
	t.NoError(err)

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height,
			round:  ib.INITBallotFactV0.round - 1,
		},
		previousBlock: lastBlock.Hash(),
		previousRound: lastBlock.Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, t.localState, t.remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewSeal(ib))
}

// INIT Ballot, which,
// - ballot.Height() < local.Height() + 1
// - has INIT vp(VoteProof)
// - vp.Result == VoteProofDraw || vp.Result == VoteProofMajority
//
// ConsensusStateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testConsensusStateJoiningHandler) TestINITBallotWithINITVoteProofLowerHeight() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	cs.setCurrentRound(Round(1))
	lastBlock := t.localState.LastBlock()
	ib, err := NewINITBallotV0FromLocalState(t.remoteState, cs.currentRound(), nil)
	t.NoError(err)

	ib.INITBallotFactV0.height = t.remoteState.LastBlock().Height() - 1

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height - 1,
			round:  Round(0),
		},
		previousBlock: lastBlock.Hash(),
		previousRound: lastBlock.Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, t.localState, t.remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewSeal(ib))
}

// INIT Ballot, which,
// - ballot.Height() > local.Height() + 1
// - has INIT vp(VoteProof)
// - vp.Result == VoteProofDraw || vp.Result == VoteProofMajority
//
// ConsensusStateJoiningHandler will stop broadcasting it's INIT Ballot and
// moves to syncing.
func (t *testConsensusStateJoiningHandler) TestINITBallotWithINITVoteProofHigherHeight() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	cs.setCurrentRound(Round(1))

	lastBlock := t.localState.LastBlock()
	ib, err := NewINITBallotV0FromLocalState(t.remoteState, cs.currentRound(), nil)
	t.NoError(err)

	ib.INITBallotFactV0.height = t.remoteState.LastBlock().Height() + 3

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height,
			round:  Round(0),
		},
		previousBlock: lastBlock.Hash(),
		previousRound: lastBlock.Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, t.localState, t.remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(t.remoteState.Node().Privatekey(), nil)
	t.NoError(err)

	t.NoError(cs.NewSeal(ib))
}

// With new INIT VoteProof
// - vp.Height() == local + 1
// ConsensusStateJoiningHandler will moves to consensus state.
func (t *testConsensusStateJoiningHandler) TestINITVoteProofExpected() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: t.localState.LastBlock().Height() + 1,
			round:  Round(2), // round is not important to go
		},
		previousBlock: t.localState.LastBlock().Hash(),
		previousRound: t.localState.LastBlock().Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, t.localState, t.remoteState)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteProof(vp))

	var ctx ConsensusStateChangeContext
	select {
	case ctx = <-stateChan:
	case <-time.After(time.Millisecond * 100):
		t.NoError(xerrors.Errorf("failed to change state to syncing"))
	}

	t.Equal(ConsensusStateJoining, ctx.fromState)
	t.Equal(ConsensusStateConsensus, ctx.toState)
	t.Equal(StageINIT, ctx.voteProof.Stage())
	t.Equal(initFact, ctx.voteProof.Majority())
}

// With new INIT VoteProof
// - vp.Height() < local + 1
// ConsensusStateJoiningHandler will wait another VoteProof
func (t *testConsensusStateJoiningHandler) TestINITVoteProofLowerHeight() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: t.localState.LastBlock().Height(),
			round:  Round(2), // round is not important to go
		},
		previousBlock: t.localState.LastBlock().Hash(),
		previousRound: t.localState.LastBlock().Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, t.localState, t.remoteState)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteProof(vp))
}

// With new ACCEPT VoteProof
// - vp.Height() == local + 1
// ConsensusStateJoiningHandler will processing Proposal.
func (t *testConsensusStateJoiningHandler) TestACCEPTVoteProofExpected() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	returnedBlock, err := NewTestBlockV0(
		t.localState.LastBlock().Height()+1,
		Round(2),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	)
	t.NoError(err)
	proposalProcessor := NewDummyProposalProcessor(returnedBlock, nil)

	cs, err := NewConsensusStateJoiningHandler(t.localState, proposalProcessor)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: returnedBlock.Height(),
			round:  returnedBlock.Round(), // round is not important to go
		},
		proposal: returnedBlock.Proposal(),
		newBlock: returnedBlock.Hash(),
	}

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, t.localState, t.remoteState)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteProof(vp))
}

// With new ACCEPT VoteProof
// - vp.Height() < local + 1
// ConsensusStateJoiningHandler will wait another VoteProof
func (t *testConsensusStateJoiningHandler) TestACCEPTVoteProofLowerHeight() {
	thr, _ := NewThreshold(2, 67)
	_ = t.localState.Policy().SetThreshold(thr)
	_ = t.remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(t.localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: t.localState.LastBlock().Height(),
			round:  Round(2), // round is not important to go
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, t.localState, t.remoteState)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteProof(vp))
}

func TestConsensusStateJoiningHandler(t *testing.T) {
	suite.Run(t, new(testConsensusStateJoiningHandler))
}
