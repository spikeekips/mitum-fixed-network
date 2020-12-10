// +build mongodb

package isaac

import (
	"context"
	"testing"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

func TestProposalProcessorMongodb(t *testing.T) {
	handler := new(testProposalProcessor)
	handler.DBType = "mongodb"

	suite.Run(t, handler)
}

type testProposalProcessorWithGCache struct {
	baseTestStateHandler

	local  *Local
	remote *Local
}

func (t *testProposalProcessorWithGCache) SetupTest() {
	t.baseTestStateHandler.SetupTest()

	ls := t.locals(2)
	t.local, t.remote = ls[0], ls[1]
}

func (t *testProposalProcessorWithGCache) TestBlockOperations() {
	pm := NewProposalMaker(t.local)

	ib := t.newINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	opl := t.newOperationSeal(t.local, 1)
	ophs := make([]valuehash.Hash, len(opl.Operations()))
	var proposal ballot.ProposalV0
	{
		pr, err := pm.Proposal(ivp.Height(), ivp.Round())
		t.NoError(err)

		t.NoError(t.local.Storage().NewSeals([]seal.Seal{opl}))

		for i, op := range opl.Operations() {
			ophs[i] = op.Fact().Hash()
		}

		proposal = ballot.NewProposalV0(
			t.local.Node().Address(),
			pr.Height(),
			pr.Round(),
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
	t.NoError(err)

	bs, err := dp.ProcessACCEPT(proposal.Hash(), avp)
	t.NoError(err)

	blk = bs.Block()

	t.NoError(bs.Commit(context.Background()))
	t.NoError(dp.Done(proposal.Hash()))

	loaded, err := t.local.BlockFS().Load(blk.Height())
	t.NoError(err)

	t.compareBlock(blk, loaded)
	<-time.After(time.Second * 2)
	if st, ok := t.local.Storage().(DummyMongodbStorage); ok {
		for _, h := range ophs {
			t.True(t.local.Storage().HasOperationFact(h))

			a, _ := st.OperationFactCache().Get(h.String())
			t.NotNil(a)
		}
	}
}

func TestProposalProcessorMongodbWithGCache(t *testing.T) {
	handler := new(testProposalProcessorWithGCache)
	handler.DBType = "mongodb+gcache"

	suite.Run(t, handler)
}
