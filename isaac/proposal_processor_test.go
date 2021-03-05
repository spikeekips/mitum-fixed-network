package isaac

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

type dummyOperationProcessor struct {
	pool            *storage.Statepool
	beforeProcessed func(state.Processor) error
	afterProcessed  func(state.Processor) error
}

func (opp dummyOperationProcessor) New(pool *storage.Statepool) prprocessor.OperationProcessor {
	return dummyOperationProcessor{
		pool:            pool,
		beforeProcessed: opp.beforeProcessed,
		afterProcessed:  opp.afterProcessed,
	}
}

func (opp dummyOperationProcessor) PreProcess(op state.Processor) (state.Processor, error) {
	return op, nil
}

func (opp dummyOperationProcessor) Process(op state.Processor) error {
	if opp.beforeProcessed != nil {
		if err := opp.beforeProcessed(op); err != nil {
			return err
		}
	}

	if err := op.Process(opp.pool.Get, opp.pool.Set); err != nil {
		return err
	}

	if opp.afterProcessed == nil {
		return nil
	}

	return opp.afterProcessed(op)
}

func (opp dummyOperationProcessor) Close() error {
	return nil
}

func (opp dummyOperationProcessor) Cancel() error {
	return nil
}

type testProposalProcessor struct {
	BaseTest

	local  *Local
	remote *Local
}

func (t *testProposalProcessor) SetupTest() {
	t.BaseTest.SetupTest()

	ls := t.Locals(2)
	t.local, t.remote = ls[0], ls[1]
}

func (t *testProposalProcessor) processors() *prprocessor.Processors {
	pps := prprocessor.NewProcessors(NewDefaultProcessorNewFunc(
		t.local.Storage(),
		t.local.BlockFS(),
		t.local.Nodes(),
		t.Suffrage(t.local),
		nil,
	), nil)

	t.NoError(pps.Initialize())
	t.NoError(pps.Start())

	return pps
}

func (t *testProposalProcessor) newOperationSeal() (seal.Seal, KVOperation) {
	op, err := NewKVOperation(
		t.local.Node().Privatekey(),
		util.UUID().Bytes(),
		util.UUID().String(),
		util.UUID().Bytes(),
		TestNetworkID,
	)
	t.NoError(err)

	sl, err := operation.NewBaseSeal(t.local.Node().Privatekey(), []operation.Operation{op}, TestNetworkID)
	t.NoError(err)
	t.NoError(sl.IsValid(TestNetworkID))

	return sl, op
}

func (t *testProposalProcessor) TestPrepare() {
	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr, ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	t.NotNil(blk.Hash())
	t.True(blk.Proposal().Equal(pr.Hash()))
	t.Equal(blk.Height(), pr.Height())
	t.Equal(blk.Round(), pr.Round())
}

func (t *testProposalProcessor) TestPrepareRetry() {
	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	var called int64
	postPrepareHook := func(context.Context) error {
		if atomic.LoadInt64(&called) < 1 {
			atomic.AddInt64(&called, 1)

			return xerrors.Errorf("showme")
		}

		return nil
	}

	newFunc := NewDefaultProcessorNewFunc(
		t.local.Storage(),
		t.local.BlockFS(),
		t.local.Nodes(),
		t.Suffrage(t.local),
		nil,
	)
	pps := prprocessor.NewProcessors(
		func(proposal ballot.Proposal, initVoteproof base.Voteproof) (prprocessor.Processor, error) {
			if pp, err := newFunc(proposal, initVoteproof); err != nil {
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
	pch := pps.NewProposal(ctx, pr, ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	t.Equal(int64(1), atomic.LoadInt64(&called))
	t.NotNil(blk.Hash())
	t.True(blk.Proposal().Equal(pr.Hash()))
	t.Equal(blk.Height(), pr.Height())
	t.Equal(blk.Round(), pr.Round())
}

func (t *testProposalProcessor) TestSave() {
	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr, ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		ivp.Height(),
		ivp.Round(),
		pr.Hash(),
		blk.Hash(),
		nil,
	).Fact()

	avp, err := t.NewVoteproof(base.StageACCEPT, acceptFact, t.local, t.remote)
	t.NoError(err)

	sch := pps.Save(context.Background(), pr.Hash(), avp)
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case result := <-sch:
		t.Equal(prprocessor.Saved, pps.Current().State())
		t.NoError(result.Err)
	}

	// check storage
	m, found, err := t.local.Storage().ManifestByHeight(pr.Height())
	t.NoError(err)
	t.True(found)
	t.True(m.Proposal().Equal(pr.Hash()))

	b, err := t.local.BlockFS().Load(pr.Height())
	t.NoError(err)
	t.True(b.Proposal().Equal(pr.Hash()))
}

func (t *testProposalProcessor) TestCancelPreviousProposal() {
	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())

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

	t.NoError(t.local.Storage().NewSeals([]seal.Seal{sl}))

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	var pr ballot.ProposalV0
	{
		i, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
		t.NoError(err)

		pr = i.(ballot.ProposalV0)
	}

	var previous *DefaultProcessor
	{
		_ = pps.NewProposal(context.Background(), pr, ivp)

		<-time.After(timeout)

		t.Equal(prprocessor.Preparing, pps.Current().State())
		t.True(pr.Hash().Equal(pps.Current().Proposal().Hash()))

		previous = pps.Current().(*DefaultProcessor)
	}

	t.NoError(SignSeal(&pr, t.local)) // sign again to create the Proposal, which has different Hash

	_ = pps.NewProposal(context.Background(), pr, ivp)

	<-time.After(timeout * 2)

	t.Equal(prprocessor.Canceled, previous.State())
	t.Equal(prprocessor.Preparing, pps.Current().State())
	t.True(pr.Hash().Equal(pps.Current().Proposal().Hash()))
}

func (t *testProposalProcessor) TestOperation() {
	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())

	pps := t.processors()
	defer pps.Stop()

	// create new operation
	sl, op := t.newOperationSeal()
	err := t.local.Storage().NewSeals([]seal.Seal{sl})
	t.NoError(err)

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	pch := pps.NewProposal(context.Background(), pr, ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	t.Equal(1, len(blk.Operations()))
	bop := blk.Operations()[0]
	t.True(op.Hash().Equal(bop.Hash()))

	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		ivp.Height(),
		ivp.Round(),
		pr.Hash(),
		blk.Hash(),
		nil,
	).Fact()

	avp, err := t.NewVoteproof(base.StageACCEPT, acceptFact, t.local, t.remote)
	t.NoError(err)

	sch := pps.Save(context.Background(), pr.Hash(), avp)
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case result := <-sch:
		t.Equal(prprocessor.Saved, pps.Current().State())
		t.NoError(result.Err)
	}

	m, found, err := t.local.Storage().Manifest(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.True(blk.Hash().Equal(m.Hash()))

	ops, err := t.local.BlockFS().LoadOperations(pr.Height())
	t.NoError(err)

	t.Equal(1, len(ops))
	t.True(ops[0].Hash().Equal(bop.Hash()))
}

func (t *testProposalProcessor) TestSealsNotFound() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	var pr ballot.Proposal
	{
		sl, _ := t.newOperationSeal()
		t.NoError(t.remote.Storage().NewSeals([]seal.Seal{sl}))

		// add getSealHandler
		t.remote.Node().Channel().(*channetwork.Channel).SetGetSealHandler(
			func(hs []valuehash.Hash) ([]seal.Seal, error) {
				return []seal.Seal{sl}, nil
			},
		)

		pm := NewProposalMaker(t.remote.Node(), t.remote.Storage(), t.remote.Policy())
		pr, _ = pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	}

	for _, h := range pr.Seals() {
		_, found, err := t.local.Storage().Seal(h)
		t.False(found)
		t.Nil(err)
	}

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr, ivp)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())
	}

	for _, h := range pr.Seals() {
		_, found, err := t.local.Storage().Seal(h)
		t.True(found)
		t.Nil(err)
	}
}

func (t *testProposalProcessor) TestTimeoutPrepare() {
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

	t.NoError(t.local.Storage().NewSeals([]seal.Seal{sl}))

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())
	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	_ = t.local.Storage().NewProposal(pr)

	pps := t.processors()
	defer pps.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()
	pch := pps.NewProposal(ctx, pr, ivp)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.True(xerrors.Is(result.Err, context.DeadlineExceeded))
	}

	t.True(<-tryProcess)
}

func (t *testProposalProcessor) TestTimeoutSaveBeforeSavingStorage() {
	sl, _ := t.newOperationSeal()
	t.NoError(t.local.Storage().NewSeals([]seal.Seal{sl}))

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())
	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	_ = t.local.Storage().NewProposal(pr)

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr, ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired to prepare"))

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

	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		ivp.Height(),
		ivp.Round(),
		pr.Hash(),
		blk.Hash(),
		nil,
	).Fact()

	avp, err := t.NewVoteproof(base.StageACCEPT, acceptFact, t.local, t.remote)
	t.NoError(err)

	sch := pps.Save(ctx, pr.Hash(), avp)
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired to save"))

		return
	case result := <-sch:
		t.Equal(prprocessor.SaveFailed, pps.Current().State())
		t.True(xerrors.Is(result.Err, context.DeadlineExceeded))
	}
}

func (t *testProposalProcessor) TestTimeoutSaveAfterSaving() {
	sl, _ := t.newOperationSeal()
	t.NoError(t.local.Storage().NewSeals([]seal.Seal{sl}))

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())
	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	_ = t.local.Storage().NewProposal(pr)

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr, ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired to prepare"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.NoError(result.Err)
		blk = result.Block
	}

	current := pps.Current().(*DefaultProcessor)
	current.postSaveHook = func(ctx context.Context) error {
		// check saved
		m, found, err := t.local.Storage().ManifestByHeight(pr.Height())
		t.NoError(err)
		t.True(found)
		t.True(pr.Hash().Equal(m.Proposal()))
		t.True(blk.Hash().Equal(m.Hash()))

		b, err := t.local.BlockFS().Load(pr.Height())
		t.NoError(err)
		t.True(b.Proposal().Equal(pr.Hash()))

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second * 100):
		}

		return nil
	}

	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		ivp.Height(),
		ivp.Round(),
		pr.Hash(),
		blk.Hash(),
		nil,
	).Fact()

	avp, err := t.NewVoteproof(base.StageACCEPT, acceptFact, t.local, t.remote)
	t.NoError(err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
	defer cancel()

	sch := pps.Save(ctx, pr.Hash(), avp)
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired to save"))

		return
	case result := <-sch:
		t.Equal(prprocessor.SaveFailed, pps.Current().State())
		t.True(xerrors.Is(result.Err, context.DeadlineExceeded))
	}

	// temporary data will be removed.
	_, found, err := t.local.Storage().ManifestByHeight(pr.Height())
	t.NoError(err)
	t.False(found)

	bblk, err := t.local.BlockFS().Load(pr.Height())
	t.True(xerrors.Is(err, storage.NotFoundError))
	t.Nil(bblk)
}

func (t *testProposalProcessor) TestCustomOperationProcessor() {
	sl, _ := t.newOperationSeal()
	t.NoError(t.local.Storage().NewSeals([]seal.Seal{sl}))

	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

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
		t.local.Storage(),
		t.local.BlockFS(),
		t.local.Nodes(),
		t.Suffrage(t.local),
		hm,
	), nil)

	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr, ivp)

	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case <-pch:
		t.Equal(prprocessor.Prepared, pps.Current().State())
		t.Equal(1, int(atomic.LoadInt64(&processed)))
	}
}

func (t *testProposalProcessor) TestNotProcessedOperations() {
	var sls []seal.Seal
	var exclude valuehash.Hash
	for i := 0; i < 2; i++ {
		sl, op := t.newOperationSeal()
		if i == 1 {
			exclude = op.Fact().Hash()
		}

		sls = append(sls, sl)
	}

	t.NoError(t.local.Storage().NewSeals(sls))

	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	var processed int64
	opr := dummyOperationProcessor{
		beforeProcessed: func(op state.Processor) error {
			if fh := op.(operation.Operation).Fact().Hash(); fh.Equal(exclude) {
				return util.IgnoreError.Errorf("exclude this operation, %v", fh)
			}

			atomic.AddInt64(&processed, 1)
			return nil
		},
	}

	hm := hint.NewHintmap()
	t.NoError(hm.Add(KVOperation{}, opr))

	pps := prprocessor.NewProcessors(NewDefaultProcessorNewFunc(
		t.local.Storage(),
		t.local.BlockFS(),
		t.local.Nodes(),
		t.Suffrage(t.local),
		hm,
	), nil)

	t.NoError(pps.Initialize())
	t.NoError(pps.Start())
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr, ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	t.Equal(int64(len(sls)-1), atomic.LoadInt64(&processed))

	_ = blk.OperationsTree().Traverse(func(_ int, key, h, v []byte) (bool, error) {
		fh := valuehash.NewBytes(key)

		m, err := base.BytesToFactMode(v)
		t.NoError(err)

		if exclude.Equal(fh) {
			t.False(m&base.FInStates != 0)
		} else {
			t.True(m&base.FInStates != 0)
		}

		return true, nil
	})
}

func (t *testProposalProcessor) TestSameStateHash() {
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

	t.NoError(t.local.Storage().NewSeals(sls))

	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	pps := t.processors()
	defer pps.Stop()

	pch := pps.NewProposal(context.Background(), pr, ivp)

	var blk block.Block
	select {
	case <-time.After(time.Second * 3):
		t.NoError(xerrors.Errorf("waiting result, but expired"))

		return
	case result := <-pch:
		t.NotNil(result.Block)
		t.Equal(prprocessor.Prepared, pps.Current().State())

		blk = result.Block
	}

	// check operation(fact) is in states
	_ = blk.OperationsTree().Traverse(func(i int, key, h, v []byte) (bool, error) {
		_, found := facts[valuehash.NewBytes(key).String()]
		t.True(found)

		m, err := base.BytesToFactMode(v)
		t.NoError(err)
		t.True(m&base.FInStates != 0)

		return true, nil
	})

	t.NotNil(blk.States())

	t.Equal(2, len(blk.States()))

	stateHashes := map[string]valuehash.Hash{}
	for _, s := range blk.States() {
		stateHashes[s.Key()] = s.Hash()
	}
}

func (t *testProposalProcessor) TestHeavyOperations() {
	var n uint = 30
	t.local.Policy().SetMaxOperationsInProposal(n)
	pm := NewProposalMaker(t.local.Node(), t.local.Storage(), t.local.Policy())

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

	err := t.local.Storage().NewSeals(sls)
	t.NoError(err)

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr, err := pm.Proposal(ivp.Height(), ivp.Round(), ivp)
	t.NoError(err)

	pps := t.processors()
	defer pps.Stop()

	_ = pps.NewProposal(context.Background(), pr, ivp)

	var previous prprocessor.Processor
	for {
		<-time.After(time.Millisecond * 100)
		if previous = pps.Current(); previous.State() == prprocessor.Preparing {
			break
		}
	}
	<-time.After(time.Second * 1)

	// NOTE submit new Proposal
	var newpr ballot.Proposal
	{
		npr := pr.(ballot.ProposalV0)
		t.NoError(SignSeal(&npr, t.local))

		newpr = npr
	}

	pch := pps.NewProposal(context.Background(), newpr, ivp)

	result := <-pch
	t.NotNil(result.Block)
	t.Equal(prprocessor.Canceled, previous.State())
	t.Equal(prprocessor.Prepared, pps.Current().State())

	t.Equal(int64(n*2), atomic.LoadInt64(&operated))
}

func TestProposalProcessor(t *testing.T) {
	suite.Run(t, new(testProposalProcessor))
}
