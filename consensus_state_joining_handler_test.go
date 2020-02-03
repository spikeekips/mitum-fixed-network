package mitum

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
	localNode := RandomLocalNode("local", nil)
	localState := NewLocalState(localNode, NewLocalPolicy()).
		SetLastBlockHeight(Height(33)).
		SetLastBlockRound(Round(3)).
		SetLastBlockHash(valuehash.RandomSHA256())

	remoteNode := RandomLocalNode("remote", nil)
	remoteState := NewLocalState(remoteNode, NewLocalPolicy()).
		SetLastBlockHeight(localState.LastBlockHeight()).
		SetLastBlockRound(localState.LastBlockRound()).
		SetLastBlockHash(localState.LastBlockHash())

	t.NoError(localState.Nodes().Add(remoteNode))
	t.NoError(remoteState.Nodes().Add(localNode))

	return localState, remoteState
}

func (t *testConsensusStateJoiningHandler) newINITBallot(localState *LocalState, round Round) INITBallotV0 {
	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localState.Node().Address(),
		},
		INITBallotV0Fact: INITBallotV0Fact{
			BaseBallotV0Fact: BaseBallotV0Fact{
				height: localState.LastBlockHeight() + 1,
				round:  round,
			},
			previousBlock: localState.LastBlockHash(),
			previousRound: localState.LastBlockRound(),
		},
	}

	return ib
}

func (t *testConsensusStateJoiningHandler) newVoteProof(stage Stage, fact Fact, states ...*LocalState) (VoteProof, error) {
	factHash, err := fact.Hash(nil)
	if err != nil {
		return VoteProof{}, err
	}

	ballots := map[Address]valuehash.Hash{}
	votes := map[Address]VoteProofNodeFact{}

	for _, state := range states {
		factSignature, err := state.Node().Privatekey().Sign(factHash.Bytes())
		if err != nil {
			return VoteProof{}, err
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
	case ACCEPTBallotV0Fact:
		height = f.Height()
		round = f.Round()
	case INITBallotV0Fact:
		height = f.Height()
		round = f.Round()
	}

	vp := VoteProof{
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

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
}

func (t *testConsensusStateJoiningHandler) TestKeepBroadcastingINITBallot() {
	localState, remoteState := t.states()

	_, _ = localState.Policy().SetIntervalBroadcastingINITBallotInJoining(time.Millisecond * 30)
	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	time.Sleep(time.Millisecond * 50)

	received := <-remoteState.Node().Channel().ReceiveSeal()
	t.NotNil(received)

	t.Implements((*seal.Seal)(nil), received)
	t.IsType(INITBallotV0{}, received)

	ballot := received.(INITBallotV0)

	t.NoError(ballot.IsValid(nil))

	t.True(localState.Node().Publickey().Equal(ballot.Signer()))
	t.Equal(StageINIT, ballot.Stage())
	t.Equal(localState.LastBlockHeight()+1, ballot.Height())
	t.Equal(Round(0), ballot.Round())
	t.True(localState.Node().Address().Equal(ballot.Node()))

	t.True(localState.LastBlockHash().Equal(ballot.PreviousBlock()))
	t.Equal(localState.LastBlockRound(), ballot.PreviousRound())
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

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	ib := t.newINITBallot(remoteState, cs.currentRound())

	// ACCEPT VoteProof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ACCEPTBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: localState.LastBlockHeight(),
			round:  localState.LastBlockRound(),
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

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	ib := t.newINITBallot(remoteState, cs.currentRound())
	ib.INITBallotV0Fact.height = remoteState.LastBlockHeight() - 1

	// ACCEPT VoteProof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ACCEPTBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: ib.INITBallotV0Fact.height - 1,
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

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	ib := t.newINITBallot(remoteState, cs.currentRound())
	ib.INITBallotV0Fact.height = remoteState.LastBlockHeight() + 2
	ib.INITBallotV0Fact.previousBlock = valuehash.RandomSHA256()
	ib.INITBallotV0Fact.previousRound = Round(0)

	// ACCEPT VoteProof; 2 node(local and remote) vote with same AcceptFact.
	acceptFact := ACCEPTBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: ib.INITBallotV0Fact.height - 1,
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
	t.Equal(StageACCEPT, ctx.voteProof.stage)
	t.Equal(acceptFact, ctx.voteProof.majority)
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

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	cs.setCurrentRound(Round(1))
	ib := t.newINITBallot(remoteState, cs.currentRound())

	initFact := INITBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: ib.INITBallotV0Fact.height,
			round:  ib.INITBallotV0Fact.round - 1,
		},
		previousBlock: localState.LastBlockHash(),
		previousRound: localState.LastBlockRound(),
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

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	cs.setCurrentRound(Round(1))
	ib := t.newINITBallot(remoteState, cs.currentRound())
	ib.INITBallotV0Fact.height = remoteState.LastBlockHeight() - 1

	initFact := INITBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: ib.INITBallotV0Fact.height - 1,
			round:  Round(0),
		},
		previousBlock: localState.LastBlockHash(),
		previousRound: localState.LastBlockRound(),
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

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	cs.setCurrentRound(Round(1))
	ib := t.newINITBallot(remoteState, cs.currentRound())
	ib.INITBallotV0Fact.height = remoteState.LastBlockHeight() + 3

	initFact := INITBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: ib.INITBallotV0Fact.height,
			round:  Round(0),
		},
		previousBlock: localState.LastBlockHash(),
		previousRound: localState.LastBlockRound(),
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

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	initFact := INITBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: localState.LastBlockHeight() + 1,
			round:  Round(2), // round is not important to go
		},
		previousBlock: localState.LastBlockHash(),
		previousRound: localState.LastBlockRound(),
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
	t.Equal(StageINIT, ctx.voteProof.stage)
	t.Equal(initFact, ctx.voteProof.majority)
}

// With new INIT VoteProof
// - vp.Height() > local + 1
// ConsensusStateJoiningHandler will moves to syncing state.
func (t *testConsensusStateJoiningHandler) TestINITVoteProofHigherHeight() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	initFact := INITBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: localState.LastBlockHeight() + 3,
			round:  Round(2), // round is not important to go
		},
		previousBlock: localState.LastBlockHash(),
		previousRound: localState.LastBlockRound(),
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
	t.Equal(StageINIT, ctx.voteProof.stage)
	t.Equal(initFact, ctx.voteProof.majority)
}

// With new INIT VoteProof
// - vp.Height() < local + 1
// ConsensusStateJoiningHandler will wait another VoteProof
func (t *testConsensusStateJoiningHandler) TestINITVoteProofLowerHeight() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	initFact := INITBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: localState.LastBlockHeight(),
			round:  Round(2), // round is not important to go
		},
		previousBlock: localState.LastBlockHash(),
		previousRound: localState.LastBlockRound(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewVoteProof(vp))
}

func TestConsensusStateJoiningHandler(t *testing.T) {
	suite.Run(t, new(testConsensusStateJoiningHandler))
}
