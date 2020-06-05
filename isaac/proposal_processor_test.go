package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	channetwork "github.com/spikeekips/mitum/network/gochan"
)

type testProposalProcessor struct {
	baseTestStateHandler
}

func (t *testProposalProcessor) TestProcess() {
	pm := NewProposalMaker(t.localstate)

	ib := t.newINITBallot(t.localstate, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)
	proposal, err := pm.Proposal(ivp.Round())

	_ = t.localstate.Storage().NewProposal(proposal)

	dp := NewProposalProcessorV0(t.localstate, t.suffrage(t.localstate))

	blk, err := dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)
	t.NotNil(blk)
}

func (t *testProposalProcessor) TestBlockOperations() {
	pm := NewProposalMaker(t.localstate)

	ib := t.newINITBallot(t.localstate, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)

	var proposal ballot.ProposalV0
	{
		pr, err := pm.Proposal(ivp.Round())
		t.NoError(err)

		opl := t.newOperationSeal(t.localstate)
		t.NoError(t.localstate.Storage().NewSeals([]seal.Seal{opl}))

		proposal = ballot.NewProposalV0(
			t.localstate.Node().Address(),
			pr.Height(),
			pr.Round(),
			opl.OperationHashes(),
			[]valuehash.Hash{opl.Hash()},
		)
		t.NoError(SignSeal(&proposal, t.localstate))

		_ = t.localstate.Storage().NewProposal(proposal)
	}

	dp := NewProposalProcessorV0(t.localstate, t.suffrage(t.localstate))

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

	avp, err := t.newVoteproof(base.StageACCEPT, acceptFact, t.localstate, t.remoteState)

	bs, err := dp.ProcessACCEPT(proposal.Hash(), avp)
	t.NoError(err)
	t.NoError(bs.Commit())

	loaded, found, err := t.localstate.Storage().Block(blk.Hash())
	t.NoError(err)
	t.True(found)

	t.compareBlock(bs.Block(), loaded)
}

func (t *testProposalProcessor) TestNotFoundInProposal() {
	pm := NewProposalMaker(t.localstate)

	ib := t.newINITBallot(t.localstate, base.Round(0), nil)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(base.StageINIT, initFact, t.localstate, t.remoteState)

	var proposal ballot.ProposalV0
	{
		pr, err := pm.Proposal(ivp.Round())
		t.NoError(err)

		op := t.newOperationSeal(t.remoteState)

		// add getSealHandler
		t.remoteState.Node().Channel().(*channetwork.NetworkChanChannel).SetGetSealHandler(
			func(hs []valuehash.Hash) ([]seal.Seal, error) {
				return []seal.Seal{op}, nil
			},
		)

		proposal = ballot.NewProposalV0(
			t.remoteState.Node().Address(),
			pr.Height(),
			pr.Round(),
			op.OperationHashes(),
			[]valuehash.Hash{op.Hash()},
		)
		t.NoError(SignSeal(&proposal, t.remoteState))
	}

	for _, h := range proposal.Seals() {
		_, found, err := t.localstate.Storage().Seal(h)
		t.False(found)
		t.Nil(err)
	}

	_ = t.localstate.Storage().NewProposal(proposal)

	dp := NewProposalProcessorV0(t.localstate, t.suffrage(t.localstate))
	_, err = dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)

	// local node should have the missing seals
	for _, h := range proposal.Seals() {
		_, found, err := t.localstate.Storage().Seal(h)
		t.NoError(err)
		t.True(found)
	}
}

func TestProposalProcessor(t *testing.T) {
	suite.Run(t, new(testProposalProcessor))
}
