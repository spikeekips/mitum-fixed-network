// +build test

package isaac

import (
	"io"
	"os"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
)

func SignSeal(b seal.Signer, local *Local) error {
	return b.Sign(local.Node().Privatekey(), local.Policy().NetworkID())
}

type BaseTest struct {
	suite.Suite
	StorageSupportTest
	ls   []*Local
	Root string
}

func (t *BaseTest) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

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
	_ = t.Encs.AddHinter(block.BaseBlockDataMap{})
	_ = t.Encs.AddHinter(operation.FixedTreeNode{})
	_ = t.Encs.AddHinter(tree.BaseFixedTreeNode{})
}

func (t *BaseTest) SetupTest() {
	p, err := os.MkdirTemp("", "localfs-")
	if err != nil {
		panic(err)
	}

	t.Root = p
}

func (t *BaseTest) TearDownTest() {
	_ = os.RemoveAll(t.Root)

	t.CloseStates(t.ls...)
}

func (t *BaseTest) SetupNodes(local *Local, others []*Local) {
	var nodes []*Local = []*Local{local}
	nodes = append(nodes, others...)

	lastHeight := t.LastManifest(local.Database()).Height()

	t.GenerateBlocks(nodes, lastHeight)

	for _, st := range nodes {
		nch := st.Node().Channel().(*channetwork.Channel)

		nch.SetBlockDataMapsHandler(func(heights []base.Height) ([]block.BlockDataMap, error) {
			var bds []block.BlockDataMap
			for _, h := range heights {
				bd, found, err := st.Database().BlockDataMap(h)
				if !found {
					break
				} else if err != nil {
					return nil, err
				}

				bds = append(bds, bd)
			}

			return bds, nil
		})
		nch.SetBlockDataHandler(func(p string) (io.Reader, func() error, error) {
			if i, err := st.BlockData().FS().Open(p); err != nil {
				return nil, nil, err
			} else {
				return i, i.Close, nil
			}
		})
	}
}

func (t *BaseTest) GenerateBlocks(locals []*Local, targetHeight base.Height) {
	bg, err := NewDummyBlocksV0Generator(
		locals[0],
		targetHeight,
		t.Suffrage(locals[0], locals...),
		locals,
	)
	t.NoError(err)
	t.NoError(bg.Generate(false))
}

func (t *BaseTest) Locals(n int) []*Local {
	var ls []*Local
	for i := 0; i < n; i++ {
		lst := t.Database(t.Encs, t.JSONEnc)
		localNode := channetwork.RandomLocalNode(util.UUID().String())

		root, err := os.MkdirTemp(t.Root, "localfs-")
		t.NoError(err)

		blockData := localfs.NewBlockData(root, t.JSONEnc)
		t.NoError(blockData.Initialize())

		local, err := NewLocal(lst, blockData, localNode, TestNetworkID)
		if err != nil {
			panic(err)
		} else if err := local.Initialize(); err != nil {
			panic(err)
		}

		ls = append(ls, local)
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

	suffrage := t.Suffrage(ls[0], ls...)

	bg, err := NewDummyBlocksV0Generator(ls[0], base.Height(2), suffrage, ls)
	if err != nil {
		panic(err)
	} else if err := bg.Generate(true); err != nil {
		panic(err)
	}

	t.ls = append(t.ls, ls...)

	return ls
}

func (t *BaseTest) EmptyLocal() *Local {
	lst := t.Database(nil, nil)
	localNode := channetwork.RandomLocalNode(util.UUID().String())
	blockData := localfs.NewBlockData(t.Root, t.JSONEnc)
	t.NoError(blockData.Initialize())

	local, err := NewLocal(lst, blockData, localNode, TestNetworkID)
	t.NoError(err)

	t.NoError(local.Initialize())

	return local
}

func (t *BaseTest) CloseStates(states ...*Local) {
	for _, s := range states {
		_ = s.Database().Close()
	}
}

func (t *BaseTest) NewVoteproof(
	stage base.Stage, fact base.Fact, states ...*Local,
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
		t.Suffrage(states[0], states...).Nodes(),
		states[0].Policy().ThresholdRatio(),
		base.VoteResultMajority,
		false,
		stage,
		fact,
		[]base.Fact{fact},
		votes,
		localtime.UTCNow(),
	)

	return vp, nil
}

func (t *BaseTest) Suffrage(proposerState *Local, states ...*Local) base.Suffrage {
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

func (t *BaseTest) NewINITBallot(local *Local, round base.Round, voteproof base.Voteproof) ballot.INITBallotV0 {
	var ib ballot.INITBallotV0
	if round == 0 {
		if b, err := NewINITBallotV0Round0(local.Node(), local.Database()); err != nil {
			panic(err)
		} else {
			ib = b
		}
	} else {
		if b, err := NewINITBallotV0WithVoteproof(local.Node(), local.Database(), voteproof); err != nil {
			panic(err)
		} else {
			ib = b
		}
	}

	_ = ib.Sign(local.Node().Privatekey(), local.Policy().NetworkID())

	return ib
}

func (t *BaseTest) NewINITBallotFact(local *Local, round base.Round) ballot.INITBallotFactV0 {
	var manifest block.Manifest
	switch l, found, err := local.Database().LastManifest(); {
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

func (t *BaseTest) NewACCEPTBallot(local *Local, round base.Round, proposal, newBlock valuehash.Hash, voteproof base.Voteproof) ballot.ACCEPTBallotV0 {
	manifest := t.LastManifest(local.Database())

	ab := ballot.NewACCEPTBallotV0(
		local.Node().Address(),
		manifest.Height()+1,
		round,
		proposal,
		newBlock,
		voteproof,
	)

	if err := ab.Sign(local.Node().Privatekey(), local.Policy().NetworkID()); err != nil {
		panic(err)
	}

	return ab
}

func (t *BaseTest) NewOperations(local *Local, n uint) []operation.Operation {
	pk := local.Node().Privatekey()

	var ops []operation.Operation
	for i := uint(0); i < n; i++ {
		token := []byte("this-is-token")
		op, err := NewKVOperation(pk, token, util.UUID().String(), []byte(util.UUID().String()), TestNetworkID)
		t.NoError(err)

		ops = append(ops, op)
	}

	return ops
}

func (t *BaseTest) NewOperationSeal(local *Local, n uint) (operation.Seal, []operation.Operation) {
	pk := local.Node().Privatekey()

	ops := t.NewOperations(local, n)

	sl, err := operation.NewBaseSeal(pk, ops, TestNetworkID)
	t.NoError(err)
	t.NoError(sl.IsValid(TestNetworkID))

	return sl, ops
}

func (t *BaseTest) NewProposal(local *Local, round base.Round, seals []valuehash.Hash, voteproof base.Voteproof) ballot.Proposal {
	pr, err := NewProposalV0(local.Database(), local.Node().Address(), round, seals, voteproof)
	if err != nil {
		panic(err)
	}
	if err := SignSeal(&pr, local); err != nil {
		panic(err)
	}

	return pr
}

func (t *BaseTest) NewStateValue() state.Value {
	v, err := state.NewBytesValue(util.UUID().Bytes())
	t.NoError(err)

	return v
}

func (t *BaseTest) NewState(height base.Height) state.State {
	s, err := state.NewStateV0(util.UUID().String(), t.NewStateValue(), height)
	t.NoError(err)
	i, err := s.SetHash(s.GenerateHash())
	t.NoError(err)

	return i
}

func (t *BaseTest) CompareManifest(a, b block.Manifest) {
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

func (t *BaseTest) CompareProposal(a, b ballot.Proposal) {
	t.Equal(a.Node(), b.Node())
	t.Equal(a.Signature(), b.Signature())
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())
	t.True(localtime.Equal(a.SignedAt(), b.SignedAt()))
	t.True(a.Signer().Equal(b.Signer()))
	t.True(a.Hash().Equal(b.Hash()))
	t.True(a.BodyHash().Equal(b.BodyHash()))
	t.Equal(a.FactSignature(), b.FactSignature())
	t.True(a.Fact().Hash().Equal(b.Fact().Hash()))

	av := a.Voteproof()
	bv := b.Voteproof()
	if av == nil {
		t.Nil(bv)
	} else {
		t.NotNil(bv)

		t.CompareVoteproof(a.Voteproof(), b.Voteproof())
	}

	as := a.Seals()
	bs := b.Seals()
	for i := range as {
		t.True(as[i].Equal(bs[i]))
	}
}

func (t *BaseTest) CompareVoteproof(a, b base.Voteproof) {
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())
	t.Equal(a.ThresholdRatio(), b.ThresholdRatio())
	t.Equal(a.Result(), b.Result())
	t.Equal(a.Stage(), b.Stage())

	t.Equal(a.Majority().Bytes(), b.Majority().Bytes())
	t.Equal(len(a.Facts()), len(b.Facts()))

	af := a.Facts()
	bf := b.Facts()
	for i := range af {
		t.Equal(af[i].Bytes(), bf[i].Bytes())
	}

	t.Equal(len(a.Votes()), len(b.Votes()))
	av := a.Votes()
	bv := b.Votes()
	for i := range av {
		t.True(av[i].Fact().Equal(bv[i].Fact()))
		t.True(av[i].Signature().Equal(bv[i].Signature()))
		t.True(av[i].Signer().Equal(bv[i].Signer()))
	}
}

func (t *BaseTest) LastManifest(st storage.Database) block.Manifest {
	if m, found, err := st.LastManifest(); !found {
		panic(util.NotFoundError.Errorf("last manifest not found"))
	} else if err != nil {
		panic(err)
	} else {
		return m
	}
}

func (t *BaseTest) DummyProcessors() *prprocessor.Processors {
	pp := prprocessor.NewProcessors(
		(&prprocessor.DummyProcessor{}).New,
		nil,
	)
	t.NoError(pp.Initialize())
	t.NoError(pp.Start())

	return pp
}

func (t *BaseTest) Processors(newFunc prprocessor.ProcessorNewFunc) *prprocessor.Processors {
	pp := prprocessor.NewProcessors(newFunc, nil)

	t.NoError(pp.Initialize())
	t.NoError(pp.Start())

	return pp
}

func (t *BaseTest) Ballotbox(suffrage base.Suffrage, policy *LocalPolicy) *Ballotbox {
	return NewBallotbox(
		suffrage.Nodes,
		func() base.Threshold {
			if t, err := base.NewThreshold(
				uint(len(suffrage.Nodes())),
				policy.ThresholdRatio(),
			); err != nil {
				panic(err)
			} else {
				return t
			}
		},
	)
}
