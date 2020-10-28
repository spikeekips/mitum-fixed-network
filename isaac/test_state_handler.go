// +build test

package isaac

import (
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

type baseTestStateHandler struct {
	suite.Suite
	StorageSupportTest
	localfs.BaseTestBlocks
	ls []*Localstate
}

func (t *baseTestStateHandler) SetupSuite() {
	t.StorageSupportTest.SetupSuite()
	t.BaseTestBlocks.SetupSuite()

	_ = t.Encs.AddHinter(key.BTCPrivatekeyHinter)
	_ = t.Encs.AddHinter(key.BTCPublickeyHinter)
	_ = t.Encs.AddHinter(valuehash.SHA256{})
	_ = t.Encs.AddHinter(valuehash.Bytes{})
	_ = t.Encs.AddHinter(base.StringAddress(""))
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
	_ = t.Encs.AddHinter(block.ConsensusInfoV0{})
	_ = t.Encs.AddHinter(block.SuffrageInfoV0{})
	_ = t.Encs.AddHinter(operation.BaseFactSign{})
	_ = t.Encs.AddHinter(operation.BaseSeal{})
	_ = t.Encs.AddHinter(operation.KVOperationFact{})
	_ = t.Encs.AddHinter(operation.KVOperation{})
	_ = t.Encs.AddHinter(KVOperation{})
	_ = t.Encs.AddHinter(LongKVOperation{})
	_ = t.Encs.AddHinter(tree.FixedTree{})
	_ = t.Encs.AddHinter(state.StateV0{})
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

		blockfs := t.BlockFS(t.JSONEnc)
		localstate, err := NewLocalstate(lst, blockfs, localNode, TestNetworkID)
		if err != nil {
			panic(err)
		} else if err := localstate.Initialize(); err != nil {
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
}

func (t *baseTestStateHandler) TearDownTest() {
	t.closeStates(t.ls...)
}

func (t *baseTestStateHandler) lastINITVoteproof(localstate *Localstate) base.Voteproof {
	vp, _, _ := localstate.BlockFS().LastVoteproof(base.StageINIT)

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

	var votes []base.VoteproofNodeFact

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

		votes = append(votes, base.NewVoteproofNodeFact(
			state.Node().Address(),
			valuehash.RandomSHA256(),
			factHash,
			factSignature,
			state.Node().Publickey(),
		))
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
		t.suffrage(states[0], states...).Nodes(),
		states[0].Policy().ThresholdRatio(),
		base.VoteResultMajority,
		false,
		stage,
		fact,
		[]base.Fact{fact},
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

	sf := base.NewFixedSuffrage(proposerState.Node().Address(), nodes)

	if err := sf.Initialize(); err != nil {
		panic(err)
	}

	return sf
}

func (t *baseTestStateHandler) newINITBallot(localstate *Localstate, round base.Round, voteproof base.Voteproof) ballot.INITBallotV0 {
	var ib ballot.INITBallotV0
	if round == 0 {
		if b, err := NewINITBallotV0Round0(localstate); err != nil {
			panic(err)
		} else {
			ib = b
		}
	} else {
		if b, err := NewINITBallotV0WithVoteproof(localstate.Node().Address(), voteproof); err != nil {
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

func (t *baseTestStateHandler) newProposal(localstate *Localstate, round base.Round, seals []valuehash.Hash) ballot.Proposal {
	pr, err := NewProposalV0(localstate.Storage(), localstate.Node().Address(), round, seals)
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

func (t *baseTestStateHandler) newOperationSeal(localstate *Localstate, n uint) operation.Seal {
	pk := localstate.Node().Privatekey()

	var ops []operation.Operation
	for i := uint(0); i < n; i++ {
		token := []byte("this-is-token")
		op, err := NewKVOperation(pk, token, util.UUID().String(), []byte(util.UUID().String()), nil)
		t.NoError(err)

		ops = append(ops, op)
	}

	sl, err := operation.NewBaseSeal(pk, ops, nil)
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
	t.compareFixedTree(a.OperationsTree(), b.OperationsTree())
	t.compareFixedTree(a.StatesTree(), b.StatesTree())
	t.compareVoteproof(a.ConsensusInfo().INITVoteproof(), b.ConsensusInfo().INITVoteproof())
	t.compareVoteproof(a.ConsensusInfo().ACCEPTVoteproof(), b.ConsensusInfo().ACCEPTVoteproof())

	for i := range a.Operations() {
		ao := a.Operations()[i]
		bo := b.Operations()[i]
		t.True(ao.Hash().Equal(bo.Hash()))
		t.True(ao.Fact().Hash().Equal(bo.Fact().Hash()))
	}

	for i := range a.States() {
		ao := a.States()[i]
		bo := b.States()[i]
		t.True(ao.Hash().Equal(bo.Hash()))
	}
}

func (t *baseTestStateHandler) compareVoteproof(a, b base.Voteproof) {
	t.True(a.Hint().Equal(b.Hint()))
	t.Equal(a.IsFinished(), b.IsFinished())
	t.True(localtime.Equal(a.FinishedAt(), b.FinishedAt()))
	t.Equal(a.IsClosed(), b.IsClosed())
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())
	t.Equal(a.Stage(), b.Stage())
	t.Equal(a.Result(), b.Result())
	t.True(a.Majority().Hash().Equal(b.Majority().Hash()))
	t.Equal(a.ThresholdRatio(), b.ThresholdRatio())

	for _, aFact := range a.Facts() {
		var bFact base.Fact
		for _, f := range b.Facts() {
			if aFact.Hash().Equal(f.Hash()) {
				bFact = f
				break
			}
		}

		t.NotNil(bFact)
		t.True(aFact.Hash().Equal(bFact.Hash()))
	}

	for i := range a.Votes() {
		aFact := a.Votes()[i]

		var bFact base.VoteproofNodeFact
		for j := range b.Votes() {
			if aFact.Node().Equal(b.Votes()[j].Node()) {
				bFact = b.Votes()[j]
				break
			}
		}

		t.True(aFact.Fact().Equal(bFact.Fact()))
	}
}

func (t *baseTestStateHandler) compareFixedTree(a, b tree.FixedTree) {
	if a.IsEmpty() && b.IsEmpty() {
		return
	}

	t.True(a.Hint().Equal(b.Hint()))

	t.Equal(a.Len(), b.Len())
	t.Equal(a.Root(), b.Root())

	t.NoError(a.Traverse(func(i int, key, h, _ []byte) (bool, error) {
		t.Equal(key, b.Key(i))
		t.Equal(h, b.Hash(i))

		return true, nil
	}))
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
