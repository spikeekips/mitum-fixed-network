// +build test

package isaac

import (
	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/valuehash"
	"github.com/stretchr/testify/suite"
)

type baseTestConsensusStateHandler struct { // nolint
	suite.Suite
	sealStorage SealStorage
	localState  *LocalState
	remoteState *LocalState
}

func (t *baseTestConsensusStateHandler) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(INITBallotType, "init-ballot")
	_ = hint.RegisterType(SIGNBallotType, "sign-ballot")
	_ = hint.RegisterType(ACCEPTBallotType, "accept-ballot")
	_ = hint.RegisterType(VoteProofType, "voteproof")
}

func (t *baseTestConsensusStateHandler) states() (*LocalState, *LocalState) {
	lastBlock, err := NewTestBlockV0(Height(33), Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	localNode := RandomLocalNode("local", nil)
	localState, err := NewLocalState(nil, localNode)
	t.NoError(err)
	_ = localState.SetLastBlock(lastBlock)

	remoteNode := RandomLocalNode("remote", nil)
	remoteState, err := NewLocalState(nil, remoteNode)
	t.NoError(err)
	_ = remoteState.SetLastBlock(lastBlock)

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

	// TODO close up node's Network

	return localState, remoteState
}

func (t *baseTestConsensusStateHandler) SetupTest() {
	t.sealStorage = NewMapSealStorage()
	t.localState, t.remoteState = t.states()
}

func (t *baseTestConsensusStateHandler) newVoteProof(
	stage Stage, fact Fact, states ...*LocalState,
) (VoteProofV0, error) {
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
		ballots:    ballots,
		votes:      votes,
		finishedAt: localtime.Now(),
	}

	return vp, nil
}

func (t *baseTestConsensusStateHandler) suffrage(proposerState *LocalState, states ...*LocalState) Suffrage {
	nodes := make([]Node, len(states))
	for i, s := range states {
		nodes[i] = s.Node()
	}

	return NewFixedSuffrage(proposerState.Node(), nodes)
}

func (t *baseTestConsensusStateHandler) newINITBallot(localState *LocalState, round Round) INITBallotV0 {
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

func (t *baseTestConsensusStateHandler) proposalMaker(localState *LocalState) *ProposalMaker {
	return NewProposalMaker(localState)
}
