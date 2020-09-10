package isaac

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/state"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testProposalProcessor struct {
	baseTestStateHandler

	local  *Localstate
	remote *Localstate
}

func (t *testProposalProcessor) SetupTest() {
	t.baseTestStateHandler.SetupTest()

	ls := t.localstates(2)
	t.local, t.remote = ls[0], ls[1]
}

func (t *testProposalProcessor) TestProcess() {
	pm := NewProposalMaker(t.local)

	ib := t.newINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.local, t.remote)
	proposal, err := pm.Proposal(ivp.Round())

	_ = t.local.Storage().NewProposal(proposal)

	dp := NewDefaultProposalProcessor(t.local, t.suffrage(t.local))

	blk, err := dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)
	t.NotNil(blk)
}

func (t *testProposalProcessor) TestBlockOperations() {
	pm := NewProposalMaker(t.local)

	ib := t.newINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.local, t.remote)

	opl := t.newOperationSeal(t.local, 1)
	ophs := make([]valuehash.Hash, len(opl.Operations()))
	var proposal ballot.ProposalV0
	{
		pr, err := pm.Proposal(ivp.Round())
		t.NoError(err)

		t.NoError(t.local.Storage().NewSeals([]seal.Seal{opl}))

		for i, op := range opl.Operations() {
			ophs[i] = op.Fact().Hash()
		}

		proposal = ballot.NewProposalV0(
			t.local.Node().Address(),
			pr.Height(),
			pr.Round(),
			ophs,
			[]valuehash.Hash{opl.Hash()},
		)
		t.NoError(SignSeal(&proposal, t.local))

		_ = t.local.Storage().NewProposal(proposal)
	}

	dp := NewDefaultProposalProcessor(t.local, t.suffrage(t.local))

	blk, err := dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)

	t.NotNil(blk.Operations())
	t.NotNil(blk.States())

	acceptFact := ballot.NewACCEPTBallotV0(
		nil,
		ivp.Height(),
		ivp.Round(),
		proposal.Hash(),
		blk.Hash(),
		nil,
	).Fact()

	avp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.local, t.remote)

	bs, err := dp.ProcessACCEPT(proposal.Hash(), avp)
	t.NoError(err)
	t.NoError(bs.Commit(context.Background()))
	t.NoError(dp.Done(proposal.Hash()))

	loaded, err := t.local.BlockFS().Load(blk.Height())
	t.NoError(err)

	t.compareBlock(bs.Block(), loaded)

	<-time.After(time.Second * 2)
	if st, ok := t.local.Storage().(DummyMongodbStorage); ok {
		for _, h := range ophs {
			t.True(t.local.Storage().HasOperationFact(h))

			a, _ := st.OperationFactCache().Get(h.String())
			t.NotNil(a)
		}
	}
}

func (t *testProposalProcessor) TestNotFoundInProposal() {
	pm := NewProposalMaker(t.local)

	ib := t.newINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.local, t.remote)

	var proposal ballot.ProposalV0
	{
		pr, err := pm.Proposal(ivp.Round())
		t.NoError(err)

		sl := t.newOperationSeal(t.remote, 1)

		// add getSealHandler
		t.remote.Node().Channel().(*channetwork.Channel).SetGetSealHandler(
			func(hs []valuehash.Hash) ([]seal.Seal, error) {
				return []seal.Seal{sl}, nil
			},
		)

		ophs := make([]valuehash.Hash, len(sl.Operations()))
		for i, op := range sl.Operations() {
			ophs[i] = op.Fact().Hash()
		}

		proposal = ballot.NewProposalV0(
			t.remote.Node().Address(),
			pr.Height(),
			pr.Round(),
			ophs,
			[]valuehash.Hash{sl.Hash()},
		)
		t.NoError(SignSeal(&proposal, t.remote))
	}

	for _, h := range proposal.Seals() {
		_, found, err := t.local.Storage().Seal(h)
		t.False(found)
		t.Nil(err)
	}

	_ = t.local.Storage().NewProposal(proposal)

	dp := NewDefaultProposalProcessor(t.local, t.suffrage(t.local))
	_, err = dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)

	// local node should have the missing seals
	for _, h := range proposal.Seals() {
		_, found, err := t.local.Storage().Seal(h)
		t.NoError(err)
		t.True(found)
	}
}

func (t *testProposalProcessor) TestTimeoutProcessProposal() {
	timeout := time.Millisecond * 100
	t.local.Policy().SetTimeoutProcessProposal(timeout)

	var sls []seal.Seal
	for i := 0; i < 3; i++ {
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
				<-time.After(time.Millisecond * 500)

				return nil
			})

		sl, err := operation.NewBaseSeal(t.local.Node().Privatekey(), []operation.Operation{op}, TestNetworkID)
		t.NoError(err)
		t.NoError(sl.IsValid(TestNetworkID))

		sls = append(sls, sl)
	}

	err := t.local.Storage().NewSeals(sls)
	t.NoError(err)

	ib := t.newINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	pm := NewDummyProposalMaker(t.local, sls)
	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.local, t.remote)
	proposal, err := pm.Proposal(ivp.Round())
	t.NoError(err)

	_ = t.local.Storage().NewProposal(proposal)

	dp := NewDefaultProposalProcessor(t.local, t.suffrage(t.local))

	s := time.Now()
	_, err = dp.ProcessINIT(proposal.Hash(), ivp)
	t.Contains(err.Error(), "timeout to process Proposal")

	t.True(time.Since(s) < timeout*2)
}

type dummyOperationProcessor struct {
	pool            *Statepool
	beforeProcessed func(state.Processor) error
	afterProcessed  func(state.Processor) error
}

func (opp dummyOperationProcessor) New(pool *Statepool) OperationProcessor {
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

func (t *testProposalProcessor) TestCustomOperationProcessor() {
	var sls []seal.Seal
	for i := 0; i < 2; i++ {
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

		sls = append(sls, sl)
	}

	err := t.local.Storage().NewSeals(sls)
	t.NoError(err)

	ib := t.newINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	pm := NewDummyProposalMaker(t.local, sls)
	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.local, t.remote)
	proposal, err := pm.Proposal(ivp.Round())
	t.NoError(err)

	_ = t.local.Storage().NewProposal(proposal)

	dp := NewDefaultProposalProcessor(t.local, t.suffrage(t.local))

	var processed int64
	opr := dummyOperationProcessor{
		afterProcessed: func(_ state.Processor) error {
			atomic.AddInt64(&processed, 1)

			return nil
		},
	}

	_, err = dp.AddOperationProcessor(KVOperation{}, opr)
	t.NoError(err)

	_, err = dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)
	t.Equal(int64(len(sls)), atomic.LoadInt64(&processed))
}

func (t *testProposalProcessor) TestNotProcessedOperations() {
	var sls []seal.Seal
	var exclude valuehash.Hash
	for i := 0; i < 2; i++ {
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

		if i == 1 {
			exclude = op.Fact().Hash()
		}

		sls = append(sls, sl)
	}

	err := t.local.Storage().NewSeals(sls)
	t.NoError(err)

	ib := t.newINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	pm := NewDummyProposalMaker(t.local, sls)
	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.local, t.remote)
	proposal, err := pm.Proposal(ivp.Round())
	t.NoError(err)

	_ = t.local.Storage().NewProposal(proposal)

	dp := NewDefaultProposalProcessor(t.local, t.suffrage(t.local))

	var processed int64
	opr := dummyOperationProcessor{
		beforeProcessed: func(op state.Processor) error {
			if fh := op.(operation.Operation).Fact().Hash(); fh.Equal(exclude) {
				return state.IgnoreOperationProcessingError.Errorf("exclude this operation, %v", fh)
			}

			atomic.AddInt64(&processed, 1)
			return nil
		},
	}

	_, err = dp.AddOperationProcessor(KVOperation{}, opr)
	t.NoError(err)

	blk, err := dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)
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

	err := t.local.Storage().NewSeals(sls)
	t.NoError(err)

	ib := t.newINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	pm := NewDummyProposalMaker(t.local, sls)
	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.local, t.remote)
	proposal, err := pm.Proposal(ivp.Round())
	t.NoError(err)

	_ = t.local.Storage().NewProposal(proposal)

	dp := NewDefaultProposalProcessor(t.local, t.suffrage(t.local))
	blk, err := dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)

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

	for i := 0; i < 10; i++ {
		dp := NewDefaultProposalProcessor(t.local, t.suffrage(t.local))
		blk, err := dp.ProcessINIT(proposal.Hash(), ivp)
		t.NoError(err)

		t.NotNil(blk.States())

		t.Equal(2, len(blk.States()))

		for _, s := range blk.States() {
			t.True(stateHashes[s.Key()].Equal(s.Hash()))
		}
	}
}

func (t *testProposalProcessor) TestCheckStates() {
	process := func(ops []operation.Operation) block.Block {
		var sls []seal.Seal

		for _, op := range ops {
			sl, err := operation.NewBaseSeal(t.local.Node().Privatekey(), []operation.Operation{op}, TestNetworkID)
			t.NoError(err)
			t.NoError(sl.IsValid(TestNetworkID))

			sls = append(sls, sl)
		}

		err := t.local.Storage().NewSeals(sls)
		t.NoError(err)

		ib := t.newINITBallot(t.local, base.Round(0), nil)
		initFact := ib.INITBallotFactV0

		pm := NewDummyProposalMaker(t.local, sls)
		ivp, err := t.newVoteproof(base.StageINIT, initFact, t.local, t.remote)
		proposal, err := pm.Proposal(ivp.Round())
		t.NoError(err)

		_ = t.local.Storage().NewProposal(proposal)

		dp := NewDefaultProposalProcessor(t.local, t.suffrage(t.local))

		blk, err := dp.ProcessINIT(proposal.Hash(), ivp)
		t.NoError(err)

		acceptFact := ballot.NewACCEPTBallotV0(
			nil,
			ivp.Height(),
			ivp.Round(),
			proposal.Hash(),
			blk.Hash(),
			nil,
		).Fact()

		avp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.local, t.remote)

		bs, err := dp.ProcessACCEPT(proposal.Hash(), avp)
		t.NoError(err)
		t.NoError(bs.Commit(context.Background()))
		t.NoError(dp.Done(proposal.Hash()))

		return blk
	}

	var ops []operation.Operation
	for i := 0; i < 3; i++ {
		op, err := NewKVOperation(
			t.local.Node().Privatekey(),
			util.UUID().Bytes(),
			util.UUID().String(),
			util.UUID().Bytes(),
			TestNetworkID,
		)
		t.NoError(err)

		ops = append(ops, op)
	}

	blk := process(ops)
	for _, s := range blk.States() {
		t.Equal(base.NilHeight, s.PreviousHeight())
		t.Equal(blk.Height(), s.Height())
	}

	// process same operations for next block
	var newOps []operation.Operation
	for _, op := range ops {
		op, err := NewKVOperation(
			t.local.Node().Privatekey(),
			util.UUID().Bytes(),
			op.(KVOperation).Key(),
			util.UUID().Bytes(),
			TestNetworkID,
		)
		t.NoError(err)

		newOps = append(newOps, op)
	}

	nextBlk := process(newOps)
	for _, s := range nextBlk.States() {
		t.Equal(nextBlk.Height(), s.Height())
	}
}

func TestProposalProcessor(t *testing.T) {
	suite.Run(t, new(testProposalProcessor))
}
