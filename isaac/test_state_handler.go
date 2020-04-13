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
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/localtime"
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
	_ = t.encs.AddHinter(base.NewShortAddress(""))
	_ = t.encs.AddHinter(ballot.INITBallotV0{})
	_ = t.encs.AddHinter(ballot.INITBallotFactV0{})
	_ = t.encs.AddHinter(ballot.ProposalV0{})
	_ = t.encs.AddHinter(ballot.ProposalFactV0{})
	_ = t.encs.AddHinter(ballot.SIGNBallotV0{})
	_ = t.encs.AddHinter(ballot.SIGNBallotFactV0{})
	_ = t.encs.AddHinter(ballot.ACCEPTBallotV0{})
	_ = t.encs.AddHinter(ballot.ACCEPTBallotFactV0{})
	_ = t.encs.AddHinter(base.VoteproofV0{})
	_ = t.encs.AddHinter(block.BlockV0{})
	_ = t.encs.AddHinter(block.ManifestV0{})
	_ = t.encs.AddHinter(block.BlockConsensusInfoV0{})
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
	lastBlock, err := block.NewTestBlockV0(base.Height(2), base.Round(9), nil, valuehash.RandomSHA256())
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
