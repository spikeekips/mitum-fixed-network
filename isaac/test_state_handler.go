// +build test

package isaac

import (
	"bytes"
	"sort"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/base/tree"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/localtime"
)

type baseTestStateHandler struct {
	suite.Suite
	StorageSupportTest
	localstate  *Localstate
	remoteState *Localstate
	encs        *encoder.Encoders
	enc         encoder.Encoder
	ls          []*Localstate
}

func (t *baseTestStateHandler) SetupSuite() {
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
	_ = t.Encs.AddHinter(base.BaseNodeV0{})
	_ = t.Encs.AddHinter(block.BlockV0{})
	_ = t.Encs.AddHinter(block.ManifestV0{})
	_ = t.Encs.AddHinter(block.BlockConsensusInfoV0{})
	_ = t.Encs.AddHinter(block.SuffrageInfoV0{})
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

func (t *baseTestStateHandler) localstates(n int) []*Localstate {
	var ls []*Localstate
	for i := 0; i < n; i++ {
		lst := t.Storage(t.Encs, t.JSONEnc)
		localNode := RandomLocalNode(util.UUID().String(), nil)
		localstate, err := NewLocalstate(lst, localNode, TestNetworkID)
		if err != nil {
			panic(err)
		}

		ls = append(ls, localstate)
	}

	for _, l := range ls {
		for _, r := range ls {
			if l.Node().Address() == r.Node().Address() {
				continue
			}

			if err := l.Nodes().Add(r.Node()); err != nil {
				panic(err)
			}
		}
	}

	suffrage := t.suffrage(ls[0], ls...)

	if bg, err := NewDummyBlocksV0Generator(ls[0], base.Height(2), suffrage, ls); err != nil {
		panic(err)
	} else if err := bg.Generate(true); err != nil {
		panic(err)
	}

	t.ls = append(t.ls, ls...)

	return ls
}

func (t *baseTestStateHandler) SetupTest() {
	ls := t.localstates(2)
	t.localstate, t.remoteState = ls[0], ls[1]
}

func (t *baseTestStateHandler) TearDownTest() {
	t.closeStates(t.ls...)
}

func (t *baseTestStateHandler) lastINITVoteproof(localstate *Localstate) base.Voteproof {
	vp, _, _ := localstate.Storage().LastVoteproof(base.StageINIT)

	return vp
}

func (t *baseTestStateHandler) closeStates(states ...*Localstate) {
	for _, s := range states {
		_ = s.Storage().Close()
	}
}

func (t *baseTestStateHandler) newVoteproof(
	stage base.Stage, fact base.Fact, states ...*Localstate,
) (base.VoteproofV0, error) {
	factHash := fact.Hash()

	ballots := map[base.Address]valuehash.Hash{}
	votes := map[base.Address]base.VoteproofNodeFact{}

	for _, state := range states {
		factSignature, err := state.Node().Privatekey().Sign(
			util.ConcatBytesSlice(
				factHash.Bytes(),
				state.Policy().NetworkID(),
			),
		)
		if err != nil {
			return base.VoteproofV0{}, err
		}

		ballots[state.Node().Address()] = valuehash.RandomSHA256()
		votes[state.Node().Address()] = base.NewVoteproofNodeFact(
			state.Node().Address(),
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
	nodes := make([]base.Address, len(states))
	for i, s := range states {
		nodes[i] = s.Node().Address()
	}

	return base.NewFixedSuffrage(proposerState.Node().Address(), nodes)
}

func (t *baseTestStateHandler) newINITBallot(localstate *Localstate, round base.Round, voteproof base.Voteproof) ballot.INITBallotV0 {
	var ib ballot.INITBallotV0
	if round == 0 {
		if b, err := NewINITBallotV0Round0(localstate.Storage(), localstate.Node().Address()); err != nil {
			panic(err)
		} else {
			ib = b
		}
	} else {
		if b, err := NewINITBallotV0WithVoteproof(
			localstate.Storage(),
			localstate.Node().Address(),
			voteproof,
		); err != nil {
			panic(err)
		} else {
			ib = b
		}
	}

	_ = ib.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID())

	return ib
}

func (t *baseTestStateHandler) newINITBallotFact(localstate *Localstate, round base.Round) ballot.INITBallotFactV0 {
	var manifest block.Manifest
	switch l, found, err := localstate.Storage().LastManifest(); {
	case !found:
		panic(xerrors.Errorf("last block not found: %w", err))
	case err != nil:
		panic(xerrors.Errorf("failed to get last block: %w", err))
	default:
		manifest = l
	}

	return ballot.NewINITBallotFactV0(
		manifest.Height()+1,
		round,
		manifest.Hash(),
	)
}

func (t *baseTestStateHandler) newProposal(localstate *Localstate, round base.Round, operations, seals []valuehash.Hash) ballot.Proposal {
	pr, err := NewProposalV0(localstate.Storage(), localstate.Node().Address(), round, operations, seals)
	if err != nil {
		panic(err)
	}
	if err := SignSeal(&pr, localstate); err != nil {
		panic(err)
	}

	return pr
}

func (t *baseTestStateHandler) newACCEPTBallot(localstate *Localstate, round base.Round, proposal, newBlock valuehash.Hash) ballot.ACCEPTBallotV0 {
	manifest := t.lastManifest(localstate.Storage())

	ab := ballot.NewACCEPTBallotV0(
		localstate.Node().Address(),
		manifest.Height()+1,
		round,
		proposal,
		newBlock,
		nil,
	)

	if err := ab.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID()); err != nil {
		panic(err)
	}

	return ab
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
	if a.OperationsHash() == nil {
		t.Nil(b.OperationsHash())
	} else {
		t.True(a.OperationsHash().Equal(b.OperationsHash()))
	}

	if a.StatesHash() == nil {
		t.Nil(b.StatesHash())
	} else {
		t.True(a.StatesHash().Equal(b.StatesHash()))
	}
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

func (t *baseTestStateHandler) lastManifest(st storage.Storage) block.Manifest {
	if m, found, err := st.LastManifest(); !found {
		panic(storage.NotFoundError.Errorf("last manifest not found"))
	} else if err != nil {
		panic(err)
	} else {
		return m
	}
}

func (t *baseTestStateHandler) lastBlock(st storage.Storage) block.Block {
	if m, found, err := st.LastBlock(); !found {
		panic(storage.NotFoundError.Errorf("last block not found"))
	} else if err != nil {
		panic(err)
	} else {
		return m
	}
}
