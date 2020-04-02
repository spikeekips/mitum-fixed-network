package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/valuehash"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testProposalProcessor struct {
	baseTestStateHandler
}

func (t *testProposalProcessor) TestProcess() {
	pm := NewProposalMaker(t.localstate)

	ib, err := NewINITBallotV0FromLocalstate(t.localstate, Round(0))
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(StageINIT, initFact, t.localstate, t.remoteState)
	proposal, err := pm.Proposal(ivp.Round())

	_ = t.localstate.Storage().NewProposal(proposal)

	dp := NewProposalProcessorV0(t.localstate)

	block, err := dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)
	t.NotNil(block)
}

func (t *testProposalProcessor) TestBlockOperations() {
	pm := NewProposalMaker(t.localstate)

	ib, err := NewINITBallotV0FromLocalstate(t.localstate, Round(0))
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(StageINIT, initFact, t.localstate, t.remoteState)

	var proposal Proposal
	{
		pr, err := pm.Proposal(ivp.Round())
		t.NoError(err)

		opl := t.newOperationSeal(t.localstate)
		t.NoError(t.localstate.Storage().NewSeals([]seal.Seal{opl}))

		newpr, err := NewProposal(
			t.localstate,
			pr.Height(),
			pr.Round(),
			opl.OperationHashes(),
			[]valuehash.Hash{opl.Hash()},
			t.localstate.Policy().NetworkID(),
		)
		t.NoError(err)

		proposal = newpr
		_ = t.localstate.Storage().NewProposal(proposal)
	}

	dp := NewProposalProcessorV0(t.localstate)

	block, err := dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)

	t.NotNil(block.Operations())
	t.NotNil(block.States())

	acceptFact := ACCEPTBallotFactV0{
		BaseBallotFactV0: BaseBallotFactV0{
			height: ivp.Height(),
			round:  ivp.Round(),
		},
		proposal: proposal.Hash(),
		newBlock: block.Hash(),
	}

	avp, err := t.newVoteproof(StageACCEPT, acceptFact, t.localstate, t.remoteState)

	bs, err := dp.ProcessACCEPT(proposal.Hash(), avp)
	t.NoError(err)
	t.NoError(bs.Commit())

	loaded, err := t.localstate.Storage().Block(block.Hash())
	t.NoError(err)

	t.compareBlock(bs.Block(), loaded)
}

func (t *testProposalProcessor) TestNotFoundInProposal() {
	pm := NewProposalMaker(t.localstate)

	ib, err := NewINITBallotV0FromLocalstate(t.localstate, Round(0))
	t.NoError(err)
	initFact := ib.INITBallotFactV0

	ivp, err := t.newVoteproof(StageINIT, initFact, t.localstate, t.remoteState)

	var proposal Proposal
	{
		pr, err := pm.Proposal(ivp.Round())
		t.NoError(err)

		op := t.newOperationSeal(t.remoteState)

		// add getSealHandler
		t.remoteState.Node().Channel().(*NetworkChanChannel).SetGetSealHandler(
			func(hs []valuehash.Hash) ([]seal.Seal, error) {
				return []seal.Seal{op}, nil
			},
		)

		newpr, err := NewProposal(
			t.remoteState,
			pr.Height(),
			pr.Round(),
			op.OperationHashes(),
			[]valuehash.Hash{op.Hash()},
			t.remoteState.Policy().NetworkID(),
		)
		t.NoError(err)

		proposal = newpr
	}

	for _, h := range proposal.Seals() {
		_, err = t.localstate.Storage().Seal(h)
		t.True(xerrors.Is(err, storage.NotFoundError))
	}

	_ = t.localstate.Storage().NewProposal(proposal)

	dp := NewProposalProcessorV0(t.localstate)
	_, err = dp.ProcessINIT(proposal.Hash(), ivp)
	t.NoError(err)

	// local node should have the missing seals
	for _, h := range proposal.Seals() {
		_, err = t.localstate.Storage().Seal(h)
		t.NoError(err)
	}
}

func TestProposalProcessor(t *testing.T) {
	suite.Run(t, new(testProposalProcessor))
}
