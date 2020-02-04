package mitum

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

type testConsensusStateConsensusHandler struct {
	suite.Suite

	policy *LocalPolicy
}

func (t *testConsensusStateConsensusHandler) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(INITBallotType, "init-ballot")
	_ = hint.RegisterType(SIGNBallotType, "sign-ballot")
	_ = hint.RegisterType(ACCEPTBallotType, "accept-ballot")
}

func (t *testConsensusStateConsensusHandler) states() (*LocalState, *LocalState) {
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

	// TODO close up node's Network

	return localState, remoteState
}

func (t *testConsensusStateConsensusHandler) newINITBallot(localState *LocalState, round Round) INITBallotV0 {
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

func (t *testConsensusStateConsensusHandler) newVoteProof(stage Stage, fact Fact, states ...*LocalState) (VoteProof, error) {
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

func (t *testConsensusStateConsensusHandler) TestNew() {
	localState, remoteState := t.states()
	localState.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	cs, err := NewConsensusStateConsensusHandler(localState)
	t.NoError(err)
	t.NotNil(cs)

	initFact := INITBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: localState.LastBlockHeight() + 1,
			round:  Round(0),
		},
		previousBlock: localState.LastBlockHash(),
		previousRound: localState.LastBlockRound(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
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

func (t *testConsensusStateConsensusHandler) TestWaitingProposal() {
	localState, remoteState := t.states()
	localState.Policy().SetTimeoutWaitingProposal(time.Millisecond * 3)
	localState.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 5)

	cs, err := NewConsensusStateConsensusHandler(localState)
	t.NoError(err)
	t.NotNil(cs)
	_ = cs.SetLogger(*log) // TODO remove

	initFact := INITBallotV0Fact{
		BaseBallotV0Fact: BaseBallotV0Fact{
			height: localState.LastBlockHeight() + 1,
			round:  Round(0),
		},
		previousBlock: localState.LastBlockHash(),
		previousRound: localState.LastBlockRound(),
	}

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(ConsensusStateChangeContext{
		fromState: ConsensusStateJoining,
		toState:   ConsensusStateJoining,
		voteProof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	<-time.After(time.Millisecond * 10)

	for i := 0; i < 2; i++ {
		r := <-remoteState.Node().Channel().ReceiveSeal()
		t.NotNil(r)

		rb := r.(INITBallotV0)

		t.Equal(StageINIT, rb.Stage())
		t.Equal(vp.Height(), rb.Height())
		t.Equal(vp.Round()+1, rb.Round()) // means that handler moves to next round
	}
}

func TestConsensusStateConsensusHandler(t *testing.T) {
	suite.Run(t, new(testConsensusStateConsensusHandler))
}
