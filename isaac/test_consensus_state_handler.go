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

type baseTestStateHandler struct { // nolint
	suite.Suite
	sealStorage SealStorage
	localstate  *Localstate
	remoteState *Localstate
}

func (t *baseTestStateHandler) SetupSuite() {
	_ = hint.RegisterType(key.BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(key.BTCPublickey{}.Hint().Type(), "btc-publickey")
	_ = hint.RegisterType(valuehash.SHA256{}.Hint().Type(), "sha256")
	_ = hint.RegisterType(encoder.JSONEncoder{}.Hint().Type(), "json-encoder")
	_ = hint.RegisterType((NewShortAddress("")).Hint().Type(), "short-address")
	_ = hint.RegisterType(INITBallotType, "init-ballot")
	_ = hint.RegisterType(SIGNBallotType, "sign-ballot")
	_ = hint.RegisterType(ACCEPTBallotType, "accept-ballot")
	_ = hint.RegisterType(VoteproofType, "voteproof")
}

func (t *baseTestStateHandler) states() (*Localstate, *Localstate) {
	lastBlock, err := NewTestBlockV0(Height(33), Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	localNode := RandomLocalNode("local", nil)
	localstate, err := NewLocalstate(nil, localNode)
	t.NoError(err)
	_ = localstate.SetLastBlock(lastBlock)

	remoteNode := RandomLocalNode("remote", nil)
	remoteState, err := NewLocalstate(nil, remoteNode)
	t.NoError(err)
	_ = remoteState.SetLastBlock(lastBlock)

	t.NoError(localstate.Nodes().Add(remoteNode))
	t.NoError(remoteState.Nodes().Add(localNode))

	lastINITVoteproof := NewDummyVoteproof(
		localstate.LastBlock().Height(),
		localstate.LastBlock().Round(),
		StageINIT,
		VoteproofMajority,
	)
	_ = localstate.SetLastINITVoteproof(lastINITVoteproof)
	_ = remoteState.SetLastINITVoteproof(lastINITVoteproof)
	lastACCEPTVoteproof := NewDummyVoteproof(
		localstate.LastBlock().Height(),
		localstate.LastBlock().Round(),
		StageACCEPT,
		VoteproofMajority,
	)
	_ = localstate.SetLastACCEPTVoteproof(lastACCEPTVoteproof)
	_ = remoteState.SetLastACCEPTVoteproof(lastACCEPTVoteproof)

	// TODO close up node's Network

	return localstate, remoteState
}

func (t *baseTestStateHandler) SetupTest() {
	t.sealStorage = NewMapSealStorage()
	t.localstate, t.remoteState = t.states()
}

func (t *baseTestStateHandler) newVoteproof(
	stage Stage, fact Fact, states ...*Localstate,
) (VoteproofV0, error) {
	factHash, err := fact.Hash(nil)
	if err != nil {
		return VoteproofV0{}, err
	}

	ballots := map[Address]valuehash.Hash{}
	votes := map[Address]VoteproofNodeFact{}

	for _, state := range states {
		factSignature, err := state.Node().Privatekey().Sign(factHash.Bytes())
		if err != nil {
			return VoteproofV0{}, err
		}

		ballots[state.Node().Address()] = valuehash.RandomSHA256()
		votes[state.Node().Address()] = VoteproofNodeFact{
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

	vp := VoteproofV0{
		height:    height,
		round:     round,
		stage:     stage,
		threshold: states[0].Policy().Threshold(),
		result:    VoteproofMajority,
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

func (t *baseTestStateHandler) suffrage(proposerState *Localstate, states ...*Localstate) Suffrage {
	nodes := make([]Node, len(states))
	for i, s := range states {
		nodes[i] = s.Node()
	}

	return NewFixedSuffrage(proposerState.Node(), nodes)
}

func (t *baseTestStateHandler) newINITBallot(localstate *Localstate, round Round) INITBallotV0 {
	ib := INITBallotV0{
		BaseBallotV0: BaseBallotV0{
			node: localstate.Node().Address(),
		},
		INITBallotFactV0: INITBallotFactV0{
			BaseBallotFactV0: BaseBallotFactV0{
				height: localstate.LastBlock().Height() + 1,
				round:  round,
			},
			previousBlock: localstate.LastBlock().Hash(),
			previousRound: localstate.LastBlock().Round(),
		},
	}

	return ib
}

func (t *baseTestStateHandler) proposalMaker(localstate *Localstate) *ProposalMaker {
	return NewProposalMaker(localstate)
}
