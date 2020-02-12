package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

type testConsensusStates struct {
	suite.Suite

	policy *LocalPolicy
}

func (t *testConsensusStates) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(INITBallotType, "init-ballot")
}

func (t *testConsensusStates) states() (*LocalState, *LocalState) {
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

func (t *testConsensusStates) newVoteProof(stage Stage, fact Fact, states ...*LocalState) (VoteProofV0, error) {
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

func (t *testConsensusStates) TestINITVoteProofHigherHeight() {
	localState, remoteState := t.states()

	thr, _ := NewThreshold(2, 67)
	_ = localState.Policy().SetThreshold(thr)
	_ = remoteState.Policy().SetThreshold(thr)

	css := NewConsensusStates(localState, nil, nil, nil, nil, nil, nil, nil, nil)
	t.NotNil(css)

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

	t.NoError(css.newVoteProof(vp))

	ctx := <-css.stateChan

	t.Equal(ConsensusStateSyncing, ctx.toState)
	t.Equal(StageINIT, ctx.voteProof.Stage())
	t.Equal(initFact, ctx.voteProof.Majority())
}

func TestConsensusStates(t *testing.T) {
	suite.Run(t, new(testConsensusStates))
}
