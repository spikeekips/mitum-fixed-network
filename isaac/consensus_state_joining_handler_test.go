package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type testConsensusStateJoiningHandler struct {
	suite.Suite

	policy *LocalPolicy
}

func (t *testConsensusStateJoiningHandler) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(INITBallotType, "init-ballot")
}

func (t *testConsensusStateJoiningHandler) states() (*LocalState, *LocalState) {
	lastBlock, err := NewTestBlockV0(Height(33), Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	localNode := RandomLocalNode("local", nil)
	localState := NewLocalState(localNode, NewLocalPolicy()).
		SetLastBlock(lastBlock)

	remoteNode := RandomLocalNode("remote", nil)
	remoteState := NewLocalState(remoteNode, NewLocalPolicy()).
		SetLastBlock(lastBlock)

	t.NoError(localState.Nodes().Add(remoteNode))
	t.NoError(remoteState.Nodes().Add(localNode))

	lastINITVoteProof := NewDummyVoteProof(
		localState.LastBlock().Height(),
		localState.LastBlock().Round(),
		StageINIT,
		VoteProofMajority,
	)
	_ = localState.SetLastINITVoteProof(lastINITVoteProof)
	_ = remoteState.SetLastINITVoteProof(lastINITVoteProof)
	lastACCEPTVoteProof := NewDummyVoteProof(
		localState.LastBlock().Height(),
		localState.LastBlock().Round(),
		StageACCEPT,
		VoteProofMajority,
	)
	_ = localState.SetLastACCEPTVoteProof(lastACCEPTVoteProof)
	_ = remoteState.SetLastACCEPTVoteProof(lastACCEPTVoteProof)

	return localState, remoteState
}

func (t *testConsensusStateJoiningHandler) newINITBallot(localState *LocalState, round Round) INITBallotV0 {
	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localState.Node().Address(),
		},
		INITBallotFactV0: INITBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: localState.LastBlock().Height() + 1,
				round:  round,
			},
			previousBlock: localState.LastBlock().Hash(),
			previousRound: localState.LastBlock().Round(),
		},
	}

	return ib
}

func (t *testConsensusStateJoiningHandler) newVoteProof(stage Stage, fact Fact, states ...*LocalState) (VoteProofV0, error) {
	factHash, err := fact.Hash(nil)
	if err != nil {
		return VoteProofV0{}, err
	}

	ballots := map[Address]valuehash.Hash{}
	votes := map[Address]VoteProofNodeFact{}

	for _, state := range states {
		factSignature, err := state.Node().Privatekey().Sign(factHash.Bytes())
		if err != nil {
			return VoteProofV0{}, err
		}

		ballots[state.Node().Address()] = valuehash.RandomSHA256()
		votes[state.Node().Address()] = VoteProofNodeFact{
			fact:          factHash,
			factSignature: factSignature,
			signer:        state.Node().Publickey(),
		}
	}

	var height Height
	var round Round
	switch f := fact.(type) {
	case ACCEPTBallotFactV0:
		height = f.Height()
		round = f.Round()
	case INITBallotFactV0:
		height = f.Height()
		round = f.Round()
	}

	vp := VoteProofV0{
		height:    height,
		round:     round,
		stage:     stage,
		threshold: states[0].Policy().Threshold(),
		result:    VoteProofMajority,
		majority:  fact,
		facts: map[valuehash.Hash]Fact{
			factHash: fact,
		},
		ballots: ballots,
		votes:   votes,
	}

	return vp, nil
}

func (t *testConsensusStateJoiningHandler) TestNew() {
	localState, _ := t.states()

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()
}

func (t *testConsensusStateJoiningHandler) TestKeepBroadcastingINITBallot() {
	localState, _ := t.states()

	_, _ = localState.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 30)
	cs, err := NewConsensusStateJoiningHandler(localState, nil)
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

	t.True(localState.Node().Publickey().Equal(ballot.Signer()))
	t.Equal(StageINIT, ballot.Stage())
	t.Equal(localState.LastBlock().Height()+1, ballot.Height())
	t.Equal(Round(0), ballot.Round())
	t.True(localState.Node().Address().Equal(ballot.Node()))

	t.True(localState.LastBlock().Hash().Equal(ballot.PreviousBlock()))
	t.Equal(localState.LastBlock().Round(), ballot.PreviousRound())
}

// INIT Ballot, which,
// - ballot.Height() == local.Height() + 1
// - has ACCEPT vp(VoteProof)
// - vp.Result == VoteProofMajority
//
// ConsensusStateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testConsensusStateJoiningHandler) TestINITBallotWithACCEPTVoteProofExpectedHeight() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	lastBlock := localState.LastBlock()
	ib, err := NewINITBallotV0FromLocalState(localState, cs.currentRound(), nil)
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

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, localState, remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(remoteState.Node().Privatekey(), nil)
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
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	ib, err := NewINITBallotV0FromLocalState(remoteState, cs.currentRound(), nil)
	t.NoError(err)

	ib.INITBallotFactV0.height = remoteState.LastBlock().Height() - 1

	// ACCEPT VoteProof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height - 1,
			round:  Round(0),
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, localState, remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(remoteState.Node().Privatekey(), nil)
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
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	ib, err := NewINITBallotV0FromLocalState(remoteState, cs.currentRound(), nil)
	t.NoError(err)

	ib.INITBallotFactV0.height = remoteState.LastBlock().Height() + 2
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

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, localState, remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(remoteState.Node().Privatekey(), nil)
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
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	cs.setCurrentRound(Round(1))
	lastBlock := localState.LastBlock()
	ib, err := NewINITBallotV0FromLocalState(remoteState, cs.currentRound(), nil)
	t.NoError(err)

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height,
			round:  ib.INITBallotFactV0.round - 1,
		},
		previousBlock: lastBlock.Hash(),
		previousRound: lastBlock.Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(remoteState.Node().Privatekey(), nil)
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
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	cs.setCurrentRound(Round(1))
	lastBlock := localState.LastBlock()
	ib, err := NewINITBallotV0FromLocalState(remoteState, cs.currentRound(), nil)
	t.NoError(err)

	ib.INITBallotFactV0.height = remoteState.LastBlock().Height() - 1

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height - 1,
			round:  Round(0),
		},
		previousBlock: lastBlock.Hash(),
		previousRound: lastBlock.Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(remoteState.Node().Privatekey(), nil)
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
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	cs.setCurrentRound(Round(1))

	lastBlock := localState.LastBlock()
	ib, err := NewINITBallotV0FromLocalState(remoteState, cs.currentRound(), nil)
	t.NoError(err)

	ib.INITBallotFactV0.height = remoteState.LastBlock().Height() + 3

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ib.INITBallotFactV0.height,
			round:  Round(0),
		},
		previousBlock: lastBlock.Hash(),
		previousRound: lastBlock.Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
	t.NoError(err)

	ib.voteProof = vp

	err = ib.Sign(remoteState.Node().Privatekey(), nil)
	t.NoError(err)

	t.NoError(cs.NewSeal(ib))
}

// With new INIT VoteProof
// - vp.Height() == local + 1
// ConsensusStateJoiningHandler will moves to consensus state.
func (t *testConsensusStateJoiningHandler) TestINITVoteProofExpected() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: localState.LastBlock().Height() + 1,
			round:  Round(2), // round is not important to go
		},
		previousBlock: localState.LastBlock().Hash(),
		previousRound: localState.LastBlock().Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
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
// - vp.Height() > local + 1
// ConsensusStateJoiningHandler will moves to syncing state.
func (t *testConsensusStateJoiningHandler) TestINITVoteProofHigherHeight() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: localState.LastBlock().Height() + 3,
			round:  Round(2), // round is not important to go
		},
		previousBlock: localState.LastBlock().Hash(),
		previousRound: localState.LastBlock().Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
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
	t.Equal(ConsensusStateSyncing, ctx.toState)
	t.Equal(StageINIT, ctx.voteProof.Stage())
	t.Equal(initFact, ctx.voteProof.Majority())
}

// With new INIT VoteProof
// - vp.Height() < local + 1
// ConsensusStateJoiningHandler will wait another VoteProof
func (t *testConsensusStateJoiningHandler) TestINITVoteProofLowerHeight() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	initFact := INITBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: localState.LastBlock().Height(),
			round:  Round(2), // round is not important to go
		},
		previousBlock: localState.LastBlock().Hash(),
		previousRound: localState.LastBlock().Round(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteProof(vp))
}

// With new ACCEPT VoteProof
// - vp.Height() == local + 1
// ConsensusStateJoiningHandler will processing Proposal.
func (t *testConsensusStateJoiningHandler) TestACCEPTVoteProofExpected() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	returnedBlock, err := NewTestBlockV0(
		localState.LastBlock().Height()+1,
		Round(2),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	)
	t.NoError(err)
	proposalProcessor := NewDummyProposalProcessor(returnedBlock, nil)

	cs, err := NewConsensusStateJoiningHandler(localState, proposalProcessor)
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

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, localState, remoteState)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteProof(vp))
}

// With new ACCEPT VoteProof
// - vp.Height() > local + 1
// ConsensusStateJoiningHandler will moves to syncing state.
func (t *testConsensusStateJoiningHandler) TestACCEPTVoteProofHigherHeight() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: localState.LastBlock().Height() + 3,
			round:  Round(2), // round is not important to go
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, localState, remoteState)
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
	t.Equal(ConsensusStateSyncing, ctx.toState)
	t.Equal(StageACCEPT, ctx.voteProof.Stage())
	t.Equal(acceptFact, ctx.voteProof.Majority())
}

// With new ACCEPT VoteProof
// - vp.Height() < local + 1
// ConsensusStateJoiningHandler will wait another VoteProof
func (t *testConsensusStateJoiningHandler) TestACCEPTVoteProofLowerHeight() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState, nil)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate(ConsensusStateChangeContext{}))
	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: localState.LastBlock().Height(),
			round:  Round(2), // round is not important to go
		},
		proposal: valuehash.RandomSHA256(),
		newBlock: valuehash.RandomSHA256(),
	}

	vp, err := t.newVoteProof(StageACCEPT, acceptFact, localState, remoteState)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteProof(vp))
}

func TestConsensusStateJoiningHandler(t *testing.T) {
	suite.Run(t, new(testConsensusStateJoiningHandler))
}
