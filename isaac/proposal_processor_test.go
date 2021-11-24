package isaac

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/tree"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testDefaultProposalProcessor struct {
	BaseTest

	local  *Local
	remote *Local
}

func (t *testDefaultProposalProcessor) SetupTest() {
	t.BaseTest.SetupTest()

	ls := t.Locals(2)
	t.local, t.remote = ls[0], ls[1]
}

func (t *testDefaultProposalProcessor) processors() *prprocessor.Processors {
	pps := prprocessor.NewProcessors(NewDefaultProcessorNewFunc(
		t.local.Database(),
		t.local.BlockData(),
		t.local.Nodes(),
		t.Suffrage(t.local),
		nil,
	), nil)

	t.NoError(pps.Initialize())
	t.NoError(pps.Start())

	return pps
}

func (t *testDefaultProposalProcessor) TestPrepare() {
	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	t.NotNil(blk.Hash())
	t.True(blk.Proposal().Equal(pr.Fact().Hash()))
	t.Equal(blk.Height(), pr.Fact().Height())
	t.Equal(blk.Round(), pr.Fact().Round())
}

func (t *testDefaultProposalProcessor) TestPrepareRetry() {
	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	var called int64
	postPrepareHook := func(context.Context) error {
		if atomic.LoadInt64(&called) < 1 {
			atomic.AddInt64(&called, 1)

			return errors.Errorf("showme")
		}

		return nil
	}

	newFunc := NewDefaultProcessorNewFunc(
		t.local.Database(),
		t.local.BlockData(),
		t.local.Nodes(),
		t.Suffrage(t.local),
		nil,
	)
	pps := prprocessor.NewProcessors(
		func(fact base.SignedBallotFact, initVoteproof base.Voteproof) (prprocessor.Processor, error) {
			if pp, err := newFunc(fact, initVoteproof); err != nil {
				return nil, err
			} else {
				pp.(*DefaultProcessor).postPrepareHook = postPrepareHook

				return pp, nil
			}
		},
		nil)

	t.NoError(pps.Initialize())
	t.NoError(pps.Start())

	defer pps.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	pch := pps.NewProposal(ctx, pr.SignedFact(), ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	t.Equal(int64(1), atomic.LoadInt64(&called))
	t.NotNil(blk.Hash())
	t.True(blk.Proposal().Equal(pr.Fact().Hash()))
	t.Equal(blk.Height(), pr.Fact().Height())
	t.Equal(blk.Round(), pr.Fact().Round())
}

func (t *testDefaultProposalProcessor) TestSave() {
	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	acceptFact := ballot.NewACCEPTFact(
		ivp.Height(),
		ivp.Round(),
		pr.Fact().Hash(),
		blk.Hash(),
	)

	avp, err := t.NewVoteproof(base.StageACCEPT, acceptFact, t.local, t.remote)
	t.NoError(err)

	sch := pps.Save(context.Background(), pr.Fact().Hash(), avp)
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-sch:
		t.Equal(prprocessor.Saved, pps.Current().State())
		t.NoError(result.Err)
	}

	// check storage
	m, found, err := t.local.Database().ManifestByHeight(pr.Fact().Height())
	t.NoError(err)
	t.True(found)
	t.True(m.Proposal().Equal(pr.Fact().Hash()))

	found, err = t.local.BlockData().Exists(pr.Fact().Height())
	t.NoError(err)
	t.True(found)

	_, f, err := localfs.LoadData(t.local.BlockData().(*localfs.BlockData), pr.Fact().Height(), block.BlockDataManifest)
	t.NoError(err)
	t.NotNil(f)
	defer f.Close()

	b, err := io.ReadAll(f)
	t.NoError(err)

	um, err := block.DecodeManifest(b, t.JSONEnc)
	t.NoError(err)

	t.CompareManifest(blk.Manifest(), um)
}

func (t *testDefaultProposalProcessor) TestCancelPreviousProposal() {
	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())

	timeout := time.Millisecond * 100

	pps := t.processors()
	defer pps.Stop()

	// create new operation

	kop, err := NewKVOperation(
		t.local.Node().Privatekey(),
		util.UUID().Bytes(),
		util.UUID().String(),
		util.UUID().Bytes(),
		TestNetworkID,
	)
	t.NoError(err)

	op := NewLongKVOperation(kop).
		SetPreProcess(func(
			func(key string) (state.State, bool, error),
			func(valuehash.Hash, ...state.State) error,
		) error {
			<-time.After(time.Second * 100)

			return nil
		})

	sl, err := operation.NewBaseSeal(t.local.Node().Privatekey(), []operation.Operation{op}, TestNetworkID)
	t.NoError(err)
	t.NoError(sl.IsValid(TestNetworkID))

	t.NoError(t.local.Database().NewSeals([]seal.Seal{sl}))

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	var previous *DefaultProcessor
	{
		_ = pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

		<-time.After(timeout)

		t.Equal(prprocessor.Preparing, pps.Current().State())
		t.True(pr.Fact().Hash().Equal(pps.Current().Fact().Hash()))

		previous = pps.Current().(*DefaultProcessor)
	}

	{
		spr := pr.(ballot.Proposal)
		t.NoError(SignSeal(&spr, t.local)) // sign again to create the Proposal, which has different Hash
		pr = spr
	}

	var newpr base.Proposal
	var newivp base.Voteproof
	{
		ib := t.NewINITBallot(t.local, base.Round(1), ivp)
		initFact := ib.Fact()

		ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
		t.NoError(err)

		i, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
		t.NoError(err)

		newpr = i
		newivp = ivp
	}

	_ = pps.NewProposal(context.Background(), newpr.SignedFact(), newivp)

	<-time.After(timeout * 2)

	t.Equal(prprocessor.Canceled, previous.State())
	t.Equal(prprocessor.Preparing, pps.Current().State())
	t.True(newpr.Fact().Hash().Equal(pps.Current().Fact().Hash()))
}

func (t *testDefaultProposalProcessor) TestOperation() {
	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())

	pps := t.processors()
	defer pps.Stop()

	// create new operation
	sl, ops := t.NewOperationSeal(t.local, 1)
	err := t.local.Database().NewSeals([]seal.Seal{sl})
	t.NoError(err)

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	pch := pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	t.Equal(1, len(blk.Operations()))
	bop := blk.Operations()[0]
	t.True(ops[0].Hash().Equal(bop.Hash()))

	acceptFact := ballot.NewACCEPTFact(
		ivp.Height(),
		ivp.Round(),
		pr.Fact().Hash(),
		blk.Hash(),
	)

	avp, err := t.NewVoteproof(base.StageACCEPT, acceptFact, t.local, t.remote)
	t.NoError(err)

	var newBlock block.Block
	sch := pps.Save(context.Background(), pr.Fact().Hash(), avp)
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-sch:
		t.Equal(prprocessor.Saved, pps.Current().State())
		t.NoError(result.Err)

		newBlock = result.Block
		t.NotNil(newBlock)
	}

	m, found, err := t.local.Database().Manifest(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.True(blk.Hash().Equal(m.Hash()))

	nops := newBlock.Operations()

	t.Equal(1, len(nops))
	t.True(nops[0].Hash().Equal(bop.Hash()))
}

func (t *testDefaultProposalProcessor) TestSealsNotFound() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	var pr base.Proposal
	{
		sl, _ := t.NewOperationSeal(t.local, 1)
		t.NoError(t.remote.Database().NewSeals([]seal.Seal{sl}))

		// add getSealHandler
		t.remote.Channel().(*channetwork.Channel).SetGetSealHandler(
			func(hs []valuehash.Hash) ([]seal.Seal, error) {
				return []seal.Seal{sl}, nil
			},
		)

		pm := NewProposalMaker(t.remote.Node(), t.remote.Database(), t.remote.Policy())
		pr, _ = pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	}

	for _, h := range pr.Fact().Seals() {
		_, found, err := t.local.Database().Seal(h)
		t.False(found)
		t.Nil(err)
	}

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())
	}

	for _, h := range pr.Fact().Seals() {
		_, found, err := t.local.Database().Seal(h)
		t.True(found)
		t.Nil(err)
	}
}

func (t *testDefaultProposalProcessor) TestTimeoutPrepare() {
	kop, err := NewKVOperation(
		t.local.Node().Privatekey(),
		util.UUID().Bytes(),
		util.UUID().String(),
		util.UUID().Bytes(),
		TestNetworkID,
	)
	t.NoError(err)

	tryProcess := make(chan bool, 1)
	op := NewLongKVOperation(kop).
		SetPreProcess(func(
			func(key string) (state.State, bool, error),
			func(valuehash.Hash, ...state.State) error,
		) error {
			tryProcess <- true
			<-time.After(time.Second * 100)

			return nil
		})

	sl, err := operation.NewBaseSeal(t.local.Node().Privatekey(), []operation.Operation{op}, TestNetworkID)
	t.NoError(err)
	t.NoError(sl.IsValid(TestNetworkID))

	t.NoError(t.local.Database().NewSeals([]seal.Seal{sl}))

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())
	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	_ = t.local.Database().NewProposal(pr)

	pps := t.processors()
	defer pps.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()
	pch := pps.NewProposal(ctx, pr.SignedFact(), ivp)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.True(errors.Is(result.Err, context.DeadlineExceeded))
	}

	t.True(<-tryProcess)
}

func (t *testDefaultProposalProcessor) TestTimeoutSaveBeforeSavingStorage() {
	sl, _ := t.NewOperationSeal(t.local, 1)
	t.NoError(t.local.Database().NewSeals([]seal.Seal{sl}))

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())
	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	_ = t.local.Database().NewProposal(pr)

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired to prepare"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.NoError(result.Err)
		blk = result.Block
	}

	current := pps.Current().(*DefaultProcessor)
	current.preSaveHook = func(context.Context) error {
		<-time.After(time.Millisecond * 200)

		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	acceptFact := ballot.NewACCEPTFact(
		ivp.Height(),
		ivp.Round(),
		pr.Fact().Hash(),
		blk.Hash(),
	)

	avp, err := t.NewVoteproof(base.StageACCEPT, acceptFact, t.local, t.remote)
	t.NoError(err)

	sch := pps.Save(ctx, pr.Fact().Hash(), avp)
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired to save"))

		return
	case result := <-sch:
		t.Equal(prprocessor.SaveFailed, pps.Current().State())
		t.True(errors.Is(result.Err, context.DeadlineExceeded))
	}
}

func (t *testDefaultProposalProcessor) TestTimeoutSaveAfterSaving() {
	sl, _ := t.NewOperationSeal(t.local, 1)
	t.NoError(t.local.Database().NewSeals([]seal.Seal{sl}))

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())
	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	_ = t.local.Database().NewProposal(pr)

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired to prepare"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.NoError(result.Err)
		blk = result.Block
	}

	current := pps.Current().(*DefaultProcessor)
	current.postSaveHook = func(ctx context.Context) error {
		// check saved
		m, found, err := t.local.Database().ManifestByHeight(pr.Fact().Height())
		t.NoError(err)
		t.True(found)
		t.True(pr.Fact().Hash().Equal(m.Proposal()))
		t.True(blk.Hash().Equal(m.Hash()))

		found, err = t.local.BlockData().Exists(pr.Fact().Height())
		t.NoError(err)
		t.True(found)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second * 100):
		}

		return nil
	}

	acceptFact := ballot.NewACCEPTFact(
		ivp.Height(),
		ivp.Round(),
		pr.Fact().Hash(),
		blk.Hash(),
	)

	avp, err := t.NewVoteproof(base.StageACCEPT, acceptFact, t.local, t.remote)
	t.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	sch := pps.Save(ctx, pr.Fact().Hash(), avp)
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired to save"))

		return
	case result := <-sch:
		t.Equal(prprocessor.SaveFailed, pps.Current().State())
		t.True(errors.Is(result.Err, context.DeadlineExceeded))
	}

	// temporary data will be removed.
	_, found, err := t.local.Database().ManifestByHeight(pr.Fact().Height())
	t.NoError(err)
	t.False(found)

	found, err = t.local.BlockData().Exists(pr.Fact().Height())
	t.NoError(err)
	t.False(found)
}

func (t *testDefaultProposalProcessor) TestCustomOperationProcessor() {
	sl, _ := t.NewOperationSeal(t.local, 1)
	t.NoError(t.local.Database().NewSeals([]seal.Seal{sl}))

	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	var processed int64
	opr := dummyOperationProcessor{
		afterProcessed: func(_ state.Processor) error {
			atomic.AddInt64(&processed, 1)

			return nil
		},
	}

	hm := hint.NewHintmap()
	t.NoError(hm.Add(KVOperation{}, opr))

	pps := prprocessor.NewProcessors(NewDefaultProcessorNewFunc(
		t.local.Database(),
		t.local.BlockData(),
		t.local.Nodes(),
		t.Suffrage(t.local),
		hm,
	), nil)

	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case <-pch:
		t.Equal(prprocessor.Prepared, pps.Current().State())
		t.Equal(1, int(atomic.LoadInt64(&processed)))
	}
}

func (t *testDefaultProposalProcessor) TestNotProcessedOperations() {
	var sls []seal.Seal
	var exclude valuehash.Hash
	for i := 0; i < 2; i++ {
		sl, ops := t.NewOperationSeal(t.local, 1)
		if i == 1 {
			exclude = ops[0].Fact().Hash()
		}

		sls = append(sls, sl)
	}

	t.NoError(t.local.Database().NewSeals(sls))

	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	excludeError := operation.NewBaseReasonError("exclude this operation").SetData(map[string]interface{}{"fact": exclude.Bytes()})
	var processed int64
	opr := dummyOperationProcessor{
		beforeProcessed: func(op state.Processor) error {
			if fh := op.(operation.Operation).Fact().Hash(); fh.Equal(exclude) {
				return excludeError
			}

			atomic.AddInt64(&processed, 1)
			return nil
		},
	}

	hm := hint.NewHintmap()
	t.NoError(hm.Add(KVOperation{}, opr))

	pps := prprocessor.NewProcessors(NewDefaultProcessorNewFunc(
		t.local.Database(),
		t.local.BlockData(),
		t.local.Nodes(),
		t.Suffrage(t.local),
		hm,
	), nil)

	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	t.Equal(int64(len(sls)-1), atomic.LoadInt64(&processed))

	_ = blk.OperationsTree().Traverse(func(no tree.FixedTreeNode) (bool, error) {
		ono := no.(operation.FixedTreeNode)

		fh := valuehash.NewBytes(ono.Key())

		if exclude.Equal(fh) {
			t.False(ono.InState())
			t.Equal(ono.Reason().Msg(), excludeError.Msg())
			t.Equal(ono.Reason().Data(), excludeError.Data())
		} else {
			t.True(ono.InState())
		}

		return true, nil
	})
}

func (t *testDefaultProposalProcessor) TestSameStateHash() {
	var sls []seal.Seal

	var keys []string
	var values [][]byte
	for i := 0; i < 2; i++ {
		keys = append(keys, util.UUID().String())
		values = append(values, util.UUID().Bytes())
	}

	facts := map[string]valuehash.Hash{}
	for i := 0; i < 10; i++ {
		key := keys[i%2]
		value := values[i%2]

		op, err := NewKVOperation(
			t.local.Node().Privatekey(),
			util.UUID().Bytes(),
			key,
			value,
			TestNetworkID,
		)
		t.NoError(err)

		facts[op.Fact().Hash().String()] = op.Fact().Hash()

		sl, err := operation.NewBaseSeal(t.local.Node().Privatekey(), []operation.Operation{op}, TestNetworkID)
		t.NoError(err)
		t.NoError(sl.IsValid(TestNetworkID))

		sls = append(sls, sl)
	}

	t.NoError(t.local.Database().NewSeals(sls))

	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(errors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	// check operation(fact) is in states
	_ = blk.OperationsTree().Traverse(func(no tree.FixedTreeNode) (bool, error) {
		ono := no.(operation.FixedTreeNode)
		_, found := facts[valuehash.NewBytes(ono.Key()).String()]
		t.True(found)

		t.True(ono.InState())

		return true, nil
	})

	t.NotNil(blk.States())

	t.Equal(2, len(blk.States()))

	stateHashes := map[string]valuehash.Hash{}
	for _, s := range blk.States() {
		stateHashes[s.Key()] = s.Hash()
	}
}

func (t *testDefaultProposalProcessor) TestHeavyOperations() {
	var n uint = 30
	t.local.Policy().SetMaxOperationsInProposal(n)
	pm := NewProposalMaker(t.local.Node(), t.local.Database(), t.local.Policy())

	var operated int64
	sls := make([]seal.Seal, n)
	for i := uint(0); i < n; i++ {
		kop, err := NewKVOperation(
			t.local.Node().Privatekey(),
			util.UUID().Bytes(),
			util.UUID().String(),
			util.UUID().Bytes(),
			TestNetworkID,
		)
		t.NoError(err)

		i := i
		op := NewLongKVOperation(kop).
			SetPreProcess(func(
				func(key string) (state.State, bool, error),
				func(valuehash.Hash, ...state.State) error,
			) error {
				<-time.After(time.Millisecond * 300)
				atomic.AddInt64(&operated, 1)

				return nil
			})

		sl, err := operation.NewBaseSeal(t.local.Node().Privatekey(), []operation.Operation{op}, TestNetworkID)
		t.NoError(err)
		t.NoError(sl.IsValid(TestNetworkID))

		sls[i] = sl
	}

	err := t.local.Database().NewSeals(sls)
	t.NoError(err)

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	pps := t.processors()
	defer pps.Stop()

	_ = pps.NewProposal(context.Background(), pr.SignedFact(), ivp)

	var previous prprocessor.Processor
	for {
		<-time.After(time.Millisecond * 100)
		if previous = pps.Current(); previous.State() == prprocessor.Preparing {
			break
		}
	}
	<-time.After(time.Second * 1)

	// NOTE submit new Proposal
	var newpr base.Proposal
	var newivp base.Voteproof
	{
		ib := t.NewINITBallot(t.local, base.Round(1), ivp)
		initFact := ib.Fact()

		ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
		t.NoError(err)

		i, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
		t.NoError(err)

		newpr = i
		newivp = ivp
	}

	pch := pps.NewProposal(context.Background(), newpr.SignedFact(), newivp)

	result := <-pch
	t.NotNil(result.Block)
	t.Equal(prprocessor.Canceled, previous.State())
	t.Equal(prprocessor.Prepared, pps.Current().State())

	t.Equal(int64(n*2), atomic.LoadInt64(&operated))
}

func TestDefaultProposalProcessor(t *testing.T) {
	suite.Run(t, new(testDefaultProposalProcessor))
}
