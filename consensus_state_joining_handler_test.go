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
	_, _ = localState.Policy().SetIntervalBroadcastingINITBallotInJoining(time.Millisecond * 30)

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

	factHash, err := acceptFact.Hash(nil)
	t.NoError(err)
	localFactSignature, err := localState.Node().Privatekey().Sign(factHash.Bytes())
	t.NoError(err)
	remoteFactSignature, err := remoteState.Node().Privatekey().Sign(factHash.Bytes())
	t.NoError(err)

	vp := VoteProof{
		height:    acceptFact.Height(),
		round:     acceptFact.Round(),
		stage:     StageACCEPT,
		threshold: remoteState.Policy().Threshold(),
		result:    VoteProofMajority,
		majority:  acceptFact,
		facts: map[valuehash.Hash]Fact{
			factHash: acceptFact,
		},
		ballots: map[Address]valuehash.Hash{
			localState.Node().Address():  valuehash.RandomSHA256(),
			remoteState.Node().Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			localState.Node().Address(): VoteProofNodeFact{
				fact:          factHash,
				factSignature: localFactSignature,
				signer:        localState.Node().Publickey(),
			},
			remoteState.Node().Address(): VoteProofNodeFact{
				fact:          factHash,
				factSignature: remoteFactSignature,
				signer:        remoteState.Node().Publickey(),
			},
		},
	}
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
	_, _ = localState.Policy().SetIntervalBroadcastingINITBallotInJoining(time.Millisecond * 30)

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

	factHash, err := acceptFact.Hash(nil)
	t.NoError(err)
	localFactSignature, err := localState.Node().Privatekey().Sign(factHash.Bytes())
	t.NoError(err)
	remoteFactSignature, err := remoteState.Node().Privatekey().Sign(factHash.Bytes())
	t.NoError(err)

	vp := VoteProof{
		height:    acceptFact.Height(),
		round:     acceptFact.Round(),
		stage:     StageACCEPT,
		threshold: remoteState.Policy().Threshold(),
		result:    VoteProofMajority,
		majority:  acceptFact,
		facts: map[valuehash.Hash]Fact{
			factHash: acceptFact,
		},
		ballots: map[Address]valuehash.Hash{
			localState.Node().Address():  valuehash.RandomSHA256(),
			remoteState.Node().Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			localState.Node().Address(): VoteProofNodeFact{
				fact:          factHash,
				factSignature: localFactSignature,
				signer:        localState.Node().Publickey(),
			},
			remoteState.Node().Address(): VoteProofNodeFact{
				fact:          factHash,
				factSignature: remoteFactSignature,
				signer:        remoteState.Node().Publickey(),
			},
		},
	}
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
	_, _ = localState.Policy().SetIntervalBroadcastingINITBallotInJoining(time.Millisecond * 30)

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

	factHash, err := acceptFact.Hash(nil)
	t.NoError(err)
	localFactSignature, err := localState.Node().Privatekey().Sign(factHash.Bytes())
	t.NoError(err)
	remoteFactSignature, err := remoteState.Node().Privatekey().Sign(factHash.Bytes())
	t.NoError(err)

	vp := VoteProof{
		height:    acceptFact.Height(),
		round:     acceptFact.Round(),
		stage:     StageACCEPT,
		threshold: remoteState.Policy().Threshold(),
		result:    VoteProofMajority,
		majority:  acceptFact,
		facts: map[valuehash.Hash]Fact{
			factHash: acceptFact,
		},
		ballots: map[Address]valuehash.Hash{
			localState.Node().Address():  valuehash.RandomSHA256(),
			remoteState.Node().Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			localState.Node().Address(): VoteProofNodeFact{
				fact:          factHash,
				factSignature: localFactSignature,
				signer:        localState.Node().Publickey(),
			},
			remoteState.Node().Address(): VoteProofNodeFact{
				fact:          factHash,
				factSignature: remoteFactSignature,
				signer:        remoteState.Node().Publickey(),
			},
		},
	}
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
// - vp.Result == VoteProofDraw
//
// ConsensusStateJoiningHandler will ignore this ballot and keep broadcasting it's INIT Ballot.
func (t *testConsensusStateJoiningHandler) TestINITBallotWithINITVoteProofExpectedHeight() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)
	_, _ = localState.Policy().SetIntervalBroadcastingINITBallotInJoining(time.Millisecond * 30)

	cs, err := NewConsensusStateJoiningHandler(localState)
	t.NoError(err)
	t.NotNil(cs)
	_ = cs.SetLogger(log)

	t.NoError(cs.Activate())
	defer func() {
		_ = cs.Deactivate()
	}()

	cs.setCurrentRound(Round(1))
	ib := t.newINITBallot(remoteState, cs.currentRound())

	// INIT VoteProof; 2 node(local and remote) vote with same INITFact.
	initFact := INITBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: ib.INITBallotV0Fact.height,
			round:  ib.INITBallotV0Fact.round - 1,
		},
		previousBlock: localState.LastBlockHash(),
		previousRound: localState.LastBlockRound(),
	}

	factHash, err := initFact.Hash(nil)
	t.NoError(err)
	localFactSignature, err := localState.Node().Privatekey().Sign(factHash.Bytes())
	t.NoError(err)
	remoteFactSignature, err := remoteState.Node().Privatekey().Sign(factHash.Bytes())
	t.NoError(err)

	vp := VoteProof{
		height:    initFact.Height(),
		round:     initFact.Round(),
		stage:     StageINIT,
		threshold: remoteState.Policy().Threshold(),
		result:    VoteProofMajority,
		majority:  initFact,
		facts: map[valuehash.Hash]Fact{
			factHash: initFact,
		},
		ballots: map[Address]valuehash.Hash{
			localState.Node().Address():  valuehash.RandomSHA256(),
			remoteState.Node().Address(): valuehash.RandomSHA256(),
		},
		votes: map[Address]VoteProofNodeFact{
			localState.Node().Address(): VoteProofNodeFact{
				fact:          factHash,
				factSignature: localFactSignature,
				signer:        localState.Node().Publickey(),
			},
			remoteState.Node().Address(): VoteProofNodeFact{
				fact:          factHash,
				factSignature: remoteFactSignature,
				signer:        remoteState.Node().Publickey(),
			},
		},
	}
	ib.voteProof = vp

	err = ib.Sign(remoteState.Node().Privatekey(), nil)
	t.NoError(err)

	stateChan := make(chan ConsensusStateChangeContext)
	cs.SetStateChan(stateChan)

	t.NoError(cs.NewSeal(ib))
}

func TestConsensusStateJoiningHandler(t *testing.T) {
	suite.Run(t, new(testConsensusStateJoiningHandler))
}
