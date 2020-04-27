// +build test

package isaac

import (
	"bytes"
	"sort"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/localtime"
)

type baseTestStateHandler struct { // nolint
	suite.Suite
	StorageSupportTest
	localstate  *Localstate
	remoteState *Localstate
	encs        *encoder.Encoders
	enc         encoder.Encoder
}

func (t *baseTestStateHandler) SetupSuite() { // nolint
	t.StorageSupportTest.SetupSuite()

	_ = t.Encs.AddHinter(key.BTCPrivatekey{})
	_ = t.Encs.AddHinter(key.BTCPublickey{})
	_ = t.Encs.AddHinter(valuehash.SHA256{})
	_ = t.Encs.AddHinter(valuehash.Dummy{})
	_ = t.Encs.AddHinter(base.NewShortAddress(""))
	_ = t.Encs.AddHinter(ballot.INITBallotV0{})
	_ = t.Encs.AddHinter(ballot.INITBallotFactV0{})
	_ = t.Encs.AddHinter(ballot.ProposalV0{})
	_ = t.Encs.AddHinter(ballot.ProposalFactV0{})
	_ = t.Encs.AddHinter(ballot.SIGNBallotV0{})
	_ = t.Encs.AddHinter(ballot.SIGNBallotFactV0{})
	_ = t.Encs.AddHinter(ballot.ACCEPTBallotV0{})
	_ = t.Encs.AddHinter(ballot.ACCEPTBallotFactV0{})
	_ = t.Encs.AddHinter(base.VoteproofV0{})
	_ = t.Encs.AddHinter(block.BlockV0{})
	_ = t.Encs.AddHinter(block.ManifestV0{})
	_ = t.Encs.AddHinter(block.BlockConsensusInfoV0{})
	_ = t.Encs.AddHinter(operation.Seal{})
	_ = t.Encs.AddHinter(operation.KVOperationFact{})
	_ = t.Encs.AddHinter(operation.KVOperation{})
	_ = t.Encs.AddHinter(KVOperation{})
	_ = t.Encs.AddHinter(tree.AVLTree{})
	_ = t.Encs.AddHinter(operation.OperationAVLNode{})
	_ = t.Encs.AddHinter(operation.OperationAVLNodeMutable{})
	_ = t.Encs.AddHinter(state.StateV0{})
	_ = t.Encs.AddHinter(state.OperationInfoV0{})
	_ = t.Encs.AddHinter(state.StateV0AVLNode{})
	_ = t.Encs.AddHinter(state.StateV0AVLNodeMutable{})
	_ = t.Encs.AddHinter(state.BytesValue{})
	_ = t.Encs.AddHinter(state.DurationValue{})
	_ = t.Encs.AddHinter(state.HintedValue{})
	_ = t.Encs.AddHinter(state.NumberValue{})
	_ = t.Encs.AddHinter(state.SliceValue{})
	_ = t.Encs.AddHinter(state.StringValue{})
}

func (t *baseTestStateHandler) states() (*Localstate, *Localstate) {
	lastBlock, err := block.NewTestBlockV0(base.Height(2), base.Round(9), nil, valuehash.RandomSHA256())
	t.NoError(err)

	lst := t.Storage(nil, nil)
	localNode := RandomLocalNode(util.UUID().String(), nil)
	localstate, err := NewLocalstate(lst, localNode, TestNetworkID)
	t.NoError(err)
	_ = localstate.SetLastBlock(lastBlock)

	rst := t.Storage(nil, nil)
	remoteNode := RandomLocalNode(util.UUID().String(), nil)
	remoteState, err := NewLocalstate(rst, remoteNode, TestNetworkID)
	t.NoError(err)
	_ = remoteState.SetLastBlock(lastBlock)

	t.NoError(localstate.Nodes().Add(remoteNode))
	t.NoError(remoteState.Nodes().Add(localNode))

	lastINITVoteproof := base.NewDummyVoteproof(
		localstate.LastBlock().Height(),
		localstate.LastBlock().Round(),
		base.StageINIT,
		base.VoteResultMajority,
	)
	_ = localstate.SetLastINITVoteproof(lastINITVoteproof)
	_ = remoteState.SetLastINITVoteproof(lastINITVoteproof)
	lastACCEPTVoteproof := base.NewDummyVoteproof(
		localstate.LastBlock().Height(),
		localstate.LastBlock().Round(),
		base.StageACCEPT,
		base.VoteResultMajority,
	)
	_ = localstate.SetLastACCEPTVoteproof(lastACCEPTVoteproof)
	_ = remoteState.SetLastACCEPTVoteproof(lastACCEPTVoteproof)

	// TODO close up node's Network

	return localstate, remoteState
}

func (t *baseTestStateHandler) SetupTest() {
	t.localstate, t.remoteState = t.states()
}

func (t *baseTestStateHandler) TearDownTest() {
	t.closeStates(t.localstate, t.remoteState)
}

func (t *baseTestStateHandler) closeStates(states ...*Localstate) {
	for _, s := range states {
		s.Storage().Close()
	}
}

func (t *baseTestStateHandler) newVoteproof(
	stage base.Stage, fact base.Fact, states ...*Localstate,
) (base.VoteproofV0, error) {
	factHash := fact.Hash()

	ballots := map[base.Address]valuehash.Hash{}
	votes := map[base.Address]base.VoteproofNodeFact{}

	for _, state := range states {
		factSignature, err := state.Node().Privatekey().Sign(factHash.Bytes())
		if err != nil {
			return base.VoteproofV0{}, err
		}

		ballots[state.Node().Address()] = valuehash.RandomSHA256()
		votes[state.Node().Address()] = base.NewVoteproofNodeFact(
			factHash,
			factSignature,
			state.Node().Publickey(),
		)
	}

	var height base.Height
	var round base.Round
	switch f := fact.(type) {
	case ballot.ACCEPTBallotFactV0:
		height = f.Height()
		round = f.Round()
	case ballot.INITBallotFactV0:
		height = f.Height()
		round = f.Round()
	}

	vp := base.NewTestVoteproofV0(
		height,
		round,
		states[0].Policy().Threshold(),
		base.VoteResultMajority,
		false,
		stage,
		fact,
		map[valuehash.Hash]base.Fact{
			factHash: fact,
		},
		ballots,
		votes,
		localtime.Now(),
	)

	return vp, nil
}

func (t *baseTestStateHandler) suffrage(proposerState *Localstate, states ...*Localstate) base.Suffrage {
	nodes := make([]base.Node, len(states))
	for i, s := range states {
		nodes[i] = s.Node()
	}

	return base.NewFixedSuffrage(proposerState.Node(), nodes)
}

func (t *baseTestStateHandler) newINITBallot(localstate *Localstate, round base.Round) ballot.INITBallotV0 {
	ib := ballot.NewINITBallotV0(
		localstate.Node().Address(),
		localstate.LastBlock().Height()+1,
		round,
		localstate.LastBlock().Hash(),
		localstate.LastBlock().Round(),
		nil,
	)

	_ = ib.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID())

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

func (t *baseTestStateHandler) compareManifest(a, b block.Manifest) {
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())
	t.True(a.Proposal().Equal(b.Proposal()))
	t.True(a.PreviousBlock().Equal(b.PreviousBlock()))
	t.True(a.OperationsHash().Equal(b.OperationsHash()))
	t.True(a.StatesHash().Equal(b.StatesHash()))
}

func (t *baseTestStateHandler) compareBlock(a, b block.Block) {
	t.compareManifest(a, b)
	t.compareAVLTree(a.States(), b.States())
	t.compareAVLTree(a.Operations(), b.Operations())
	t.compareVoteproof(a.INITVoteproof(), b.INITVoteproof())
	t.compareVoteproof(a.ACCEPTVoteproof(), b.ACCEPTVoteproof())
}

func (t *baseTestStateHandler) compareVoteproof(a, b base.Voteproof) {
	t.True(a.Hint().Equal(b.Hint()))
	t.Equal(a.IsFinished(), b.IsFinished())
	t.Equal(localtime.Normalize(a.FinishedAt()), localtime.Normalize(b.FinishedAt()))
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
