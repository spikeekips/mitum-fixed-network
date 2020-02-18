package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type testConsensusStateConsensusHandler struct {
	suite.Suite

	policy      *LocalPolicy
	sealStorage SealStorage
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
	_ = hint.RegisterType(VoteProofType, "voteproof")
}

func (t *testConsensusStateConsensusHandler) SetupTest() {
	t.sealStorage = NewMapSealStorage()
}

func (t *testConsensusStateConsensusHandler) states() (*LocalState, *LocalState) {
	lastBlock, err := NewTestBlockV0(Height(33), Round(0), nil, valuehash.RandomSHA256())
	t.NoError(err)

	localNode := RandomLocalNode("local", nil)
	localState, err := NewLocalState(nil, localNode)
	t.NoError(err)
	localState.SetLastBlock(lastBlock)

	remoteNode := RandomLocalNode("remote", nil)
	remoteState, err := NewLocalState(nil, remoteNode)
	t.NoError(err)
	remoteState.SetLastBlock(lastBlock)

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

func (t *testConsensusStateConsensusHandler) newVoteProof(stage Stage, fact Fact, states ...*LocalState) (VoteProofV0, error) {
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

func (t *testConsensusStateConsensusHandler) suffrage(proposerState *LocalState, states ...*LocalState) Suffrage {
	var nodes []Node
	for _, s := range states {
		nodes = append(nodes, s.Node())
	}

	return NewFixedSuffrage(proposerState.Node(), nodes)
}

func (t *testConsensusStateConsensusHandler) TestNew() {
	localState, remoteState := t.states()
	localState.Policy().SetTimeoutWaitingProposal(time.Millisecond * 10)

	suffrage := t.suffrage(remoteState, localState)

	proposalMaker := NewProposalMaker(localState)
	cs, err := NewConsensusStateConsensusHandler(
		localState, DummyProposalProcessor{}, suffrage, t.sealStorage, proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	ib, err := NewINITBallotV0FromLocalState(localState, Round(0), nil)
	t.NoError(err)
	initFact := ib.INITBallotFactV0

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

func (t *testConsensusStateConsensusHandler) TestWaitingProposalButTimeedOut() {
	localState, remoteState := t.states()
	localState.Policy().SetTimeoutWaitingProposal(time.Millisecond * 3)
	localState.Policy().SetIntervalBroadcastingINITBallot(time.Millisecond * 5)

	suffrage := t.suffrage(remoteState, localState)

	proposalMaker := NewProposalMaker(localState)
	cs, err := NewConsensusStateConsensusHandler(localState, DummyProposalProcessor{}, suffrage, t.sealStorage, proposalMaker)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	ib, err := NewINITBallotV0FromLocalState(localState, Round(0), nil)
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(ConsensusStateChangeContext{
		fromState: ConsensusStateJoining,
		toState:   ConsensusStateConsensus,
		voteProof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	<-time.After(time.Millisecond * 10)

	r := <-sealChan
	t.NotNil(r)

	rb := r.(INITBallotV0)

	t.Equal(StageINIT, rb.Stage())
	t.Equal(vp.Height(), rb.Height())
	t.Equal(vp.Round()+1, rb.Round()) // means that handler moves to next round
}

// with Proposal, ACCEPTBallot will be broadcasted with newly processed
// Proposal.
func (t *testConsensusStateConsensusHandler) TestWithProposalWaitACCEPTBallot() {
	localState, remoteState := t.states()
	localState.Policy().SetWaitBroadcastingACCEPTBallot(time.Millisecond * 1)

	ib, err := NewINITBallotV0FromLocalState(localState, Round(0), nil)
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	proposalMaker := NewProposalMaker(localState)
	cs, err := NewConsensusStateConsensusHandler(
		localState,
		DummyProposalProcessor{},
		t.suffrage(remoteState, remoteState), // localnode is not in ActingSuffrage.
		t.sealStorage,
		proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(ConsensusStateChangeContext{
		fromState: ConsensusStateJoining,
		toState:   ConsensusStateConsensus,
		voteProof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	pr, err := NewProposalFromLocalState(remoteState, initFact.round, nil, nil)
	t.NoError(err)

	returnedBlock, err := NewTestBlockV0(initFact.Height(), initFact.Round(), pr.Hash(), valuehash.RandomSHA256())
	t.NoError(err)
	cs.proposalProcessor = NewDummyProposalProcessor(returnedBlock, nil)

	t.NoError(cs.NewSeal(pr))

	r := <-sealChan
	t.NotNil(r)

	rb := r.(ACCEPTBallotV0)
	t.Equal(StageACCEPT, rb.Stage())

	t.Equal(pr.Height(), rb.Height())
	t.Equal(pr.Round(), rb.Round())
	t.True(pr.Hash().Equal(rb.Proposal()))
	t.True(returnedBlock.Hash().Equal(rb.NewBlock()))
}

// with Proposal, ACCEPTBallot will be broadcasted with newly processed
// Proposal.
func (t *testConsensusStateConsensusHandler) TestWithProposalWaitSIGNBallot() {
	localState, remoteState := t.states()

	ib, err := NewINITBallotV0FromLocalState(localState, Round(0), nil)
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	proposalMaker := NewProposalMaker(localState)
	cs, err := NewConsensusStateConsensusHandler(
		localState,
		DummyProposalProcessor{},
		t.suffrage(remoteState, localState, remoteState), // localnode is not in ActingSuffrage.
		t.sealStorage,
		proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	vp, err := t.newVoteProof(StageINIT, initFact, localState, remoteState)
	t.NoError(err)

	t.NoError(cs.Activate(ConsensusStateChangeContext{
		fromState: ConsensusStateJoining,
		toState:   ConsensusStateConsensus,
		voteProof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	pr, err := NewProposalFromLocalState(remoteState, initFact.round, nil, nil)
	t.NoError(err)

	returnedBlock, err := NewTestBlockV0(initFact.Height(), initFact.Round(), pr.Hash(), valuehash.RandomSHA256())
	t.NoError(err)
	cs.proposalProcessor = NewDummyProposalProcessor(returnedBlock, nil)

	t.NoError(cs.NewSeal(pr))

	r := <-sealChan
	t.NotNil(r)

	rb := r.(SIGNBallotV0)
	t.Equal(StageSIGN, rb.Stage())

	t.Equal(pr.Height(), rb.Height())
	t.Equal(pr.Round(), rb.Round())
	t.True(pr.Hash().Equal(rb.Proposal()))
	t.True(returnedBlock.Hash().Equal(rb.NewBlock()))
}

func (t *testConsensusStateConsensusHandler) TestDraw() {
	localState, remoteState := t.states()

	proposalMaker := NewProposalMaker(localState)
	cs, err := NewConsensusStateConsensusHandler(
		localState,
		DummyProposalProcessor{},
		t.suffrage(remoteState, localState, remoteState), // localnode is not in ActingSuffrage.
		t.sealStorage,
		proposalMaker,
	)
	t.NoError(err)
	t.NotNil(cs)

	sealChan := make(chan seal.Seal)
	cs.SetSealChan(sealChan)

	var vp VoteProof
	{
		ib, err := NewINITBallotV0FromLocalState(localState, Round(0), nil)
		t.NoError(err)
		fact := ib.INITBallotFactV0

		vp, _ = t.newVoteProof(StageINIT, fact, localState, remoteState)
	}

	t.NoError(cs.Activate(ConsensusStateChangeContext{
		fromState: ConsensusStateJoining,
		toState:   ConsensusStateConsensus,
		voteProof: vp,
	}))

	defer func() {
		_ = cs.Deactivate(ConsensusStateChangeContext{})
	}()

	var drew VoteProofV0
	{
		dummyBlock, _ := NewTestBlockV0(vp.Height(), vp.Round(), valuehash.RandomSHA256(), valuehash.RandomSHA256())

		ab, err := NewACCEPTBallotV0FromLocalState(localState, vp.Round(), dummyBlock, nil)
		t.NoError(err)
		fact := ab.ACCEPTBallotFactV0

		drew, _ = t.newVoteProof(StageINIT, fact, localState, remoteState)
		drew.result = VoteProofDraw
	}

	t.NoError(cs.NewVoteProof(drew))

	r := <-sealChan
	t.NotNil(r)
	t.Implements((*INITBallot)(nil), r)

	ib := r.(INITBallotV0)
	t.Equal(StageINIT, ib.Stage())
	t.Equal(vp.Height(), ib.Height())
	t.Equal(vp.Round()+1, ib.Round())
}

func TestConsensusStateConsensusHandler(t *testing.T) {
	suite.Run(t, new(testConsensusStateConsensusHandler))
}
