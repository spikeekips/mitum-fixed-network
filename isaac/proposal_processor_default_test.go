package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/seal"
	channetwork "github.com/spikeekips/mitum/network/gochan"
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

		opl := t.newOperationSeal(t.local)
		t.NoError(t.local.Storage().NewSeals([]seal.Seal{opl}))

		proposal = ballot.NewProposalV0(
			t.local.Node().Address(),
			pr.Height(),
			pr.Round(),
			opl.OperationHashes(),
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
	t.NoError(bs.Commit())

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

		op := t.newOperationSeal(t.remote)

		// add getSealHandler
		t.remote.Node().Channel().(*channetwork.NetworkChanChannel).SetGetSealHandler(
			func(hs []valuehash.Hash) ([]seal.Seal, error) {
				return []seal.Seal{op}, nil
			},
		)

		proposal = ballot.NewProposalV0(
			t.remote.Node().Address(),
			pr.Height(),
			pr.Round(),
			op.OperationHashes(),
			[]valuehash.Hash{op.Hash()},
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
	t.local.Policy().SetTimeoutProcessProposal(1)

	pm := NewProposalMaker(t.local)

	ib := t.newINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.local, t.remote)
	proposal, err := pm.Proposal(ivp.Round())

	_ = t.local.Storage().NewProposal(proposal)

	dp := NewDefaultProposalProcessor(t.local, t.suffrage(t.local))

	_, err = dp.ProcessINIT(proposal.Hash(), ivp)
	t.Contains(err.Error(), "timeout to process Proposal")
}

func TestProposalProcessor(t *testing.T) {
	suite.Run(t, new(testProposalProcessor))
}
