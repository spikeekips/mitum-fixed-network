// +build test

package isaac

import (
	"bytes"
	"sort"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/state"
	"github.com/spikeekips/mitum/tree"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type baseTestStateHandler struct { // nolint
	suite.Suite
	localstate  *Localstate
	remoteState *Localstate
	encs        *encoder.Encoders
	enc         encoder.Encoder
}

func (t *baseTestStateHandler) SetupSuite() { // nolint
	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewJSONEncoder()
	_ = t.encs.AddEncoder(t.enc)
	_ = t.encs.AddHinter(key.BTCPrivatekey{})
	_ = t.encs.AddHinter(key.BTCPublickey{})
	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(valuehash.Dummy{})
	_ = t.encs.AddHinter(NewShortAddress(""))
	_ = t.encs.AddHinter(INITBallotV0{})
	_ = t.encs.AddHinter(INITBallotFactV0{})
	_ = t.encs.AddHinter(ProposalV0{})
	_ = t.encs.AddHinter(ProposalFactV0{})
	_ = t.encs.AddHinter(SIGNBallotV0{})
	_ = t.encs.AddHinter(SIGNBallotFactV0{})
	_ = t.encs.AddHinter(ACCEPTBallotV0{})
	_ = t.encs.AddHinter(ACCEPTBallotFactV0{})
	_ = t.encs.AddHinter(VoteproofV0{})
	_ = t.encs.AddHinter(BlockV0{})
	_ = t.encs.AddHinter(BlockManifestV0{})
	_ = t.encs.AddHinter(BlockConsensusInfoV0{})
	_ = t.encs.AddHinter(operation.Seal{})
	_ = t.encs.AddHinter(operation.KVOperationFact{})
	_ = t.encs.AddHinter(operation.KVOperation{})
	_ = t.encs.AddHinter(KVOperation{})
	_ = t.encs.AddHinter(tree.AVLTree{})
	_ = t.encs.AddHinter(operation.OperationAVLNode{})
	_ = t.encs.AddHinter(operation.OperationAVLNodeMutable{})
	_ = t.encs.AddHinter(state.StateV0{})
	_ = t.encs.AddHinter(state.OperationInfoV0{})
	_ = t.encs.AddHinter(state.StateV0AVLNode{})
	_ = t.encs.AddHinter(state.StateV0AVLNodeMutable{})
	_ = t.encs.AddHinter(state.BytesValue{})
	_ = t.encs.AddHinter(state.DurationValue{})
	_ = t.encs.AddHinter(state.HintedValue{})
	_ = t.encs.AddHinter(state.NumberValue{})
	_ = t.encs.AddHinter(state.SliceValue{})
	_ = t.encs.AddHinter(state.StringValue{})
}

func (t *baseTestStateHandler) states() (*Localstate, *Localstate) {
	lastBlock, err := NewTestBlockV0(Height(2), Round(9), nil, valuehash.RandomSHA256())
	t.NoError(err)

	lst := NewMemStorage(t.encs, t.enc)
	localNode := RandomLocalNode(util.UUID().String(), nil)
	localstate, err := NewLocalstate(lst, localNode, TestNetworkID)
	t.NoError(err)
	_ = localstate.SetLastBlock(lastBlock)

	rst := NewMemStorage(t.encs, t.enc)
	remoteNode := RandomLocalNode(util.UUID().String(), nil)
	remoteState, err := NewLocalstate(rst, remoteNode, TestNetworkID)
	t.NoError(err)
	_ = remoteState.SetLastBlock(lastBlock)

	t.NoError(localstate.Nodes().Add(remoteNode))
	t.NoError(remoteState.Nodes().Add(localNode))

	lastINITVoteproof := NewDummyVoteproof(
		localstate.LastBlock().Height(),
		localstate.LastBlock().Round(),
		StageINIT,
		VoteResultMajority,
	)
	_ = localstate.SetLastINITVoteproof(lastINITVoteproof)
	_ = remoteState.SetLastINITVoteproof(lastINITVoteproof)
	lastACCEPTVoteproof := NewDummyVoteproof(
		localstate.LastBlock().Height(),
		localstate.LastBlock().Round(),
		StageACCEPT,
		VoteResultMajority,
	)
	_ = localstate.SetLastACCEPTVoteproof(lastACCEPTVoteproof)
	_ = remoteState.SetLastACCEPTVoteproof(lastACCEPTVoteproof)

	// TODO close up node's Network

	return localstate, remoteState
}

func (t *baseTestStateHandler) SetupTest() {
	t.localstate, t.remoteState = t.states()
}

func (t *baseTestStateHandler) newVoteproof(
	stage Stage, fact operation.Fact, states ...*Localstate,
) (VoteproofV0, error) {
	factHash := fact.Hash()

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
		result:    VoteResultMajority,
		majority:  fact,
		facts: map[valuehash.Hash]operation.Fact{
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

func (t *baseTestStateHandler) newOperationSeal(localstate *Localstate) operation.Seal {
	pk := localstate.Node().Privatekey()

	token := []byte("this-is-token")
	op, err := NewKVOperation(pk, token, util.UUID().String(), []byte(util.UUID().String()), nil)
	t.NoError(err)

	sl, err := operation.NewSeal(pk, []operation.Operation{op}, nil)
	t.NoError(err)
	t.NoError(sl.IsValid(nil))

	return sl
}

func (t *baseTestStateHandler) compareBlockManifest(a, b BlockManifest) {
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())
	t.True(a.Proposal().Equal(b.Proposal()))
	t.True(a.PreviousBlock().Equal(b.PreviousBlock()))
	t.True(a.OperationsHash().Equal(b.OperationsHash()))
	t.True(a.StatesHash().Equal(b.StatesHash()))
}

func (t *baseTestStateHandler) compareBlock(a, b Block) {
	t.compareBlockManifest(a, b)
	t.compareAVLTree(a.States(), b.States())
	t.compareAVLTree(a.Operations(), b.Operations())
	t.compareVoteproof(a.INITVoteproof(), b.INITVoteproof())
	t.compareVoteproof(a.ACCEPTVoteproof(), b.ACCEPTVoteproof())
}

func (t *baseTestStateHandler) compareVoteproof(a, b Voteproof) {
	t.True(a.Hint().Equal(b.Hint()))
	t.Equal(a.IsFinished(), b.IsFinished())
	t.Equal(localtime.RFC3339(a.FinishedAt()), localtime.RFC3339(b.FinishedAt()))
	t.Equal(a.IsClosed(), b.IsClosed())
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())
	t.Equal(a.Stage(), b.Stage())
	t.Equal(a.Result(), b.Result())
	t.Equal(a.Majority(), b.Majority())
	t.Equal(a.Ballots(), b.Ballots())
	t.Equal(a.Threshold(), b.Threshold())
}

func (t *baseTestStateHandler) compareAVLTree(a, b *tree.AVLTree) {
	if a == nil && b == nil {
		return
	}

	t.True(a.Hint().Equal(b.Hint()))
	{
		ah, err := a.RootHash()
		t.NoError(err)
		bh, err := b.RootHash()
		t.NoError(err)

		t.True(ah.Equal(bh))
	}

	var nodesA, nodesB []tree.Node
	t.NoError(a.Traverse(func(node tree.Node) (bool, error) {
		nodesA = append(nodesA, node)
		return true, nil
	}))
	t.NoError(b.Traverse(func(node tree.Node) (bool, error) {
		nodesB = append(nodesB, node)
		return true, nil
	}))

	sort.Slice(nodesA, func(i, j int) bool {
		return bytes.Compare(nodesA[i].Key(), nodesA[j].Key()) < 0
	})
	sort.Slice(nodesB, func(i, j int) bool {
		return bytes.Compare(nodesB[i].Key(), nodesB[j].Key()) < 0
	})

	t.Equal(len(nodesA), len(nodesB))

	for i := range nodesA {
		t.compareAVLTreeNode(nodesA[i].Immutable(), nodesB[i].Immutable())
	}
}

func (t *baseTestStateHandler) compareAVLTreeNode(a, b tree.Node) {
	t.Equal(a.Hint(), b.Hint())
	t.Equal(a.Key(), b.Key())
	t.Equal(a.Hash(), b.Hash())
	t.Equal(a.LeftKey(), b.LeftHash())
	t.Equal(a.LeftHash(), b.LeftHash())
	t.Equal(a.RightKey(), b.RightHash())
	t.Equal(a.RightHash(), b.RightHash())
	t.Equal(a.ValueHash(), b.ValueHash())
}
