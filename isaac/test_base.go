//go:build test
// +build test

package isaac

import (
	"io"
	"os"
	"sync"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/node"
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
	"github.com/stretchr/testify/suite"
)

func SignSeal(b seal.Signer, local *Local) error {
	return b.Sign(local.Node().Privatekey(), local.Policy().NetworkID())
}

type BaseTest struct {
	sync.Mutex
	suite.Suite
	StorageSupportTest
	ls   []*Local
	Root string
}

func (t *BaseTest) SetupSuite() {
	t.StorageSupportTest.SetupSuite()

	_ = t.Encs.TestAddHinter(KVOperation{})
	_ = t.Encs.TestAddHinter(LongKVOperation{})
	_ = t.Encs.TestAddHinter(ballot.ACCEPTFactHinter)
	_ = t.Encs.TestAddHinter(ballot.ACCEPTHinter)
	_ = t.Encs.TestAddHinter(ballot.INITFactHinter)
	_ = t.Encs.TestAddHinter(ballot.INITHinter)
	_ = t.Encs.TestAddHinter(ballot.ProposalFactHinter)
	_ = t.Encs.TestAddHinter(ballot.ProposalHinter)
	_ = t.Encs.TestAddHinter(base.BallotFactSignHinter)
	_ = t.Encs.TestAddHinter(base.BaseFactSignHinter)
	_ = t.Encs.TestAddHinter(base.SignedBallotFactHinter)
	_ = t.Encs.TestAddHinter(base.StringAddressHinter)
	_ = t.Encs.TestAddHinter(base.VoteproofV0Hinter)
	_ = t.Encs.TestAddHinter(block.BaseBlockDataMapHinter)
	_ = t.Encs.TestAddHinter(block.BlockV0Hinter)
	_ = t.Encs.TestAddHinter(block.BlockConsensusInfoV0Hinter)
	_ = t.Encs.TestAddHinter(block.ManifestV0Hinter)
	_ = t.Encs.TestAddHinter(block.SuffrageInfoV0Hinter)
	_ = t.Encs.TestAddHinter(key.BasePrivatekey{})
	_ = t.Encs.TestAddHinter(key.BasePublickey{})
	_ = t.Encs.TestAddHinter(node.BaseV0Hinter)
	_ = t.Encs.TestAddHinter(operation.FixedTreeNodeHinter)
	_ = t.Encs.TestAddHinter(operation.KVOperationFact{})
	_ = t.Encs.TestAddHinter(operation.KVOperation{})
	_ = t.Encs.TestAddHinter(operation.SealHinter)
	_ = t.Encs.TestAddHinter(state.BytesValueHinter)
	_ = t.Encs.TestAddHinter(state.DurationValueHinter)
	_ = t.Encs.TestAddHinter(state.FixedTreeNodeHinter)
	_ = t.Encs.TestAddHinter(state.HintedValueHinter)
	_ = t.Encs.TestAddHinter(state.NumberValueHinter)
	_ = t.Encs.TestAddHinter(state.SliceValueHinter)
	_ = t.Encs.TestAddHinter(state.StateV0{})
	_ = t.Encs.TestAddHinter(state.StringValueHinter)
	_ = t.Encs.TestAddHinter(tree.FixedTreeHinter)
}

func (t *BaseTest) SetupTest() {
	t.Lock()
	defer t.Unlock()

	p, err := os.MkdirTemp("", "localfs-")
	if err != nil {
		panic(err)
	}

	t.Root = p
}

func (t *BaseTest) TearDownTest() {
	t.StorageSupportTest.TearDownTest()

	_ = os.RemoveAll(t.Root)

	t.CloseStates(t.ls...)
}

func (t *BaseTest) SetupNodes(local *Local, others []*Local) {
	var nodes []*Local = []*Local{local}
	nodes = append(nodes, others...)

	lastHeight := t.LastManifest(local.Database()).Height()

	t.GenerateBlocks(nodes, lastHeight)

	for _, st := range nodes {
		nch := st.Channel().(*channetwork.Channel)

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
		uid := util.UUID().String()
		no := node.RandomLocal(uid)
		ch := channetwork.RandomChannel(uid)

		root, err := os.MkdirTemp(t.Root, "localfs-")
		t.NoError(err)

		blockData := localfs.NewBlockData(root, t.JSONEnc)
		t.NoError(blockData.Initialize())

		local, err := NewLocal(lst, blockData, no, ch, TestNetworkID)
		if err != nil {
			panic(err)
		} else if err := local.Initialize(); err != nil {
			panic(err)
		}

		ls = append(ls, local)
	}

	for _, l := range ls {
		for _, r := range ls {
			if l.Node().Address().Equal(r.Node().Address()) {
				continue
			}

			if err := l.Nodes().Add(r.Node(), r.Channel()); err != nil {
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
	uid := util.UUID().String()
	no := node.RandomLocal(uid)
	ch := channetwork.RandomChannel(uid)

	blockData := localfs.NewBlockData(t.Root, t.JSONEnc)
	t.NoError(blockData.Initialize())

	local, err := NewLocal(lst, blockData, no, ch, TestNetworkID)
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
	stage base.Stage, fact base.BallotFact, states ...*Local,
) (base.VoteproofV0, error) {
	var votes []base.SignedBallotFact

	for _, state := range states {
		fs, err := base.NewBaseBallotFactSignFromFact(fact, state.Node().Address(), state.Node().Privatekey(), state.Policy().NetworkID())
		if err != nil {
			return base.VoteproofV0{}, err
		}

		votes = append(votes, base.NewBaseSignedBallotFact(fact, fs))
	}

	var height base.Height
	var round base.Round
	switch f := fact.(type) {
	case base.ACCEPTBallotFact:
		height = f.Height()
		round = f.Round()
	case base.INITBallotFact:
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
		[]base.BallotFact{fact},
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

func (t *BaseTest) NewINITBallot(local *Local, round base.Round, voteproof base.Voteproof) base.INITBallot {
	if round == 0 {
		ib, err := NewINITBallotRound0(local.Node().Address(), local.Database(), local.Node().Privatekey(), local.Policy().NetworkID())
		if err != nil {
			panic(err)
		}

		return ib
	}

	ib, err := NewINITBallotWithVoteproof(local.Node().Address(), local.Database(), voteproof, local.Node().Privatekey(), local.Policy().NetworkID())
	if err != nil {
		panic(err)
	}

	return ib
}

func (t *BaseTest) NewINITBallotFact(local *Local, round base.Round) base.INITBallotFact {
	var manifest block.Manifest
	switch l, found, err := local.Database().LastManifest(); {
	case err != nil:
		panic(errors.Wrap(err, "failed to get last block"))
	case !found:
		panic(errors.Errorf("last block not found"))
	default:
		manifest = l
	}

	return ballot.NewINITFact(
		manifest.Height()+1,
		round,
		manifest.Hash(),
	)
}

func (t *BaseTest) NewACCEPTBallot(local *Local, round base.Round, proposal, newBlock valuehash.Hash, voteproof base.Voteproof) base.ACCEPTBallot {
	manifest := t.LastManifest(local.Database())

	ab, err := ballot.NewACCEPT(
		ballot.NewACCEPTFact(
			manifest.Height()+1,
			round,
			proposal,
			newBlock,
		),
		local.Node().Address(),
		voteproof,
		local.Node().Privatekey(), local.Policy().NetworkID(),
	)
	if err != nil {
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

func (t *BaseTest) NewProposal(local *Local, round base.Round, seals []valuehash.Hash, voteproof base.Voteproof) base.Proposal {
	var manifest block.Manifest
	switch l, found, err := local.Database().LastManifest(); {
	case err != nil:
		panic(err)
	case !found:
		panic(util.NotFoundError.Errorf("last manifest not found for NewProposalV0"))
	default:
		manifest = l
	}

	pr, err := ballot.NewProposal(
		ballot.NewProposalFact(manifest.Height()+1, round, local.Node().Address(), seals),
		local.Node().Address(),
		voteproof,
		local.Node().Privatekey(), local.Policy().NetworkID(),
	)
	if err != nil {
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

func (t *BaseTest) CompareProposal(a, b base.Proposal) {
	af := a.Fact()
	bf := b.Fact()
	afs := a.FactSign()
	bfs := b.FactSign()

	t.Equal(af.Height(), bf.Height())
	t.Equal(af.Round(), bf.Round())
	t.True(af.Hash().Equal(bf.Hash()))
	t.True(af.Hash().Equal(bf.Hash()))
	t.Equal(afs.Node(), bfs.Node())
	t.True(localtime.Equal(afs.SignedAt(), bfs.SignedAt()))
	t.True(afs.Signer().Equal(bfs.Signer()))
	t.Equal(afs.Signature(), bfs.Signature())
	t.Equal(afs.Signature(), bfs.Signature())
	t.True(a.BodyHash().Equal(b.BodyHash()))

	av := a.BaseVoteproof()
	bv := b.BaseVoteproof()
	if av == nil {
		t.Nil(bv)
	} else {
		t.NotNil(bv)

		t.CompareVoteproof(av, bv)
	}

	as := af.Operations()
	bs := bf.Operations()
	for i := range as {
		t.True(as[i].Equal(bs[i]))
	}
}

func (t *BaseTest) CompareProposalFact(a, b base.ProposalFact) {
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())
	t.True(a.Hash().Equal(b.Hash()))
	t.True(a.Hash().Equal(b.Hash()))

	as := a.Operations()
	bs := b.Operations()
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

	t.True(a.Majority().Hash().Equal(b.Majority().Hash()))
	t.True(a.Majority().Hint().Equal(b.Majority().Hint()))
	t.Equal(len(a.Facts()), len(b.Facts()))

	afs := a.Facts()
	bfs := b.Facts()
	for i := range afs {
		af := afs[i]
		bf := bfs[i]
		t.True(af.Hash().Equal(bf.Hash()))
		t.True(af.Hint().Equal(bf.Hint()))
	}

	t.Equal(len(a.Votes()), len(b.Votes()))
	av := a.Votes()
	bv := b.Votes()
	for i := range av {
		t.True(av[i].Fact().Hash().Equal(bv[i].Fact().Hash()))
		t.True(av[i].FactSign().Signature().Equal(bv[i].FactSign().Signature()))
		t.True(av[i].FactSign().Signer().Equal(bv[i].FactSign().Signer()))
	}
}

func (t *BaseTest) LastManifest(db storage.Database) block.Manifest {
	if m, found, err := db.LastManifest(); !found {
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
