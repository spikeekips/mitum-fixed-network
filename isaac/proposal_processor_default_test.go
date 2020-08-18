package isaac

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
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

	var proposal ballot.ProposalV0
	{
		pr, err := pm.Proposal(ivp.Round())
		t.NoError(err)

		opl := t.newOperationSeal(t.local, 1)
		t.NoError(t.local.Storage().NewSeals([]seal.Seal{opl}))

		ophs := make([]valuehash.Hash, len(opl.Operations()))
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

	loaded, found, err := t.local.Storage().Block(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.compareBlock(bs.Block(), loaded)
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
				func(key string) (state.StateUpdater, bool, error),
				func(valuehash.Hash, ...state.StateUpdater) error,
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
	pool           *Statepool
	afterProcessed func(dummyOperationProcessor) error
}

func (opp dummyOperationProcessor) New(pool *Statepool) OperationProcessor {
	return dummyOperationProcessor{
		pool:           pool,
		afterProcessed: opp.afterProcessed,
	}
}

func (opp dummyOperationProcessor) Process(op state.StateProcessor) error {
	if err := op.Process(opp.pool.Get, opp.pool.Set); err != nil {
		return err
	}

	if opp.afterProcessed == nil {
		return nil
	}

	return opp.afterProcessed(opp)
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
		afterProcessed: func(dummyOperationProcessor) error {
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

func TestProposalProcessor(t *testing.T) {
	suite.Run(t, new(testProposalProcessor))
}
