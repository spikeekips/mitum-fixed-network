package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
)

type testBallotChecker struct {
	baseTestStateHandler

	suf base.Suffrage
}

func (t *testBallotChecker) SetupTest() {
	t.baseTestStateHandler.SetupTest()

	t.suf = t.suffrage(t.remoteState, t.localstate)
}

func (t *testBallotChecker) TestNew() {
	t.True(t.suf.IsInside(t.localstate.Node().Address()))

	ib, err := NewINITBallotV0FromLocalstate(t.localstate, base.Round(0))
	t.NoError(err)

	bc := NewBallotChecker(ib, t.localstate, t.suf)
	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckIsInSuffrage,
	}).Check()
	t.NoError(err)
}

func (t *testBallotChecker) TestIsInSuffrage() {
	{ // from localstate
		t.True(t.suf.IsInside(t.localstate.Node().Address()))

		ib, err := NewINITBallotV0FromLocalstate(t.localstate, base.Round(0))
		t.NoError(err)

		bc := NewBallotChecker(ib, t.localstate, t.suf)

		var finished bool
		err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.CheckIsInSuffrage,
			func() (bool, error) {
				finished = true

				return true, nil
			},
		}).Check()
		t.NoError(err)

		t.True(finished)
	}

	{ // from unknown
		unknown, _ := t.states()
		t.False(t.suf.IsInside(unknown.Node().Address()))

		ib, err := NewINITBallotV0FromLocalstate(unknown, base.Round(0))
		t.NoError(err)

		bc := NewBallotChecker(ib, t.localstate, t.suf)

		var finished bool
		err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.CheckIsInSuffrage,
			func() (bool, error) {
				finished = true

				return true, nil
			},
		}).Check()
		t.NoError(err)

		t.False(finished)
	}
}

func (t *testBallotChecker) TestCheckWithLastBlock() {
	ivp := t.localstate.LastACCEPTVoteproof()

	{ // same height and next round
		ib, err := NewINITBallotV0FromLocalstate(t.localstate, ivp.Round()+1)
		t.NoError(err)

		bc := NewBallotChecker(ib, t.localstate, t.suf)

		var finished bool
		err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.CheckWithLastBlock,
			func() (bool, error) {
				finished = true

				return true, nil
			},
		}).Check()
		t.NoError(err)

		t.True(finished)
	}

	{ // lower Height
		lastBlock := t.localstate.LastBlock()
		t.NotNil(lastBlock)

		ib := ballot.NewINITBallotV0(
			t.localstate.Node().Address(),
			lastBlock.Height(),
			base.Round(0),
			lastBlock.Hash(),
			lastBlock.Round(),
			t.localstate.LastACCEPTVoteproof(),
		)

		t.NoError(ib.Sign(t.localstate.Node().Privatekey(), t.localstate.Policy().NetworkID()))

		bc := NewBallotChecker(ib, t.localstate, t.suf)

		var finished bool
		err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.CheckWithLastBlock,
			func() (bool, error) {
				finished = true

				return true, nil
			},
		}).Check()
		t.NoError(err)

		t.False(finished)
	}
}

func (t *testBallotChecker) TestCheckInvalidProposal() {
	var proposal ballot.Proposal
	{
		pr := ballot.NewProposalV0(
			t.localstate.Node().Address(),
			t.localstate.LastBlock().Height(), // wrong height
			base.Round(0),
			nil, nil,
		)

		// signed by unknown node
		pk, _ := key.NewBTCPrivatekey()
		_ = pr.Sign(pk, t.localstate.Policy().NetworkID())
		t.NoError(t.localstate.Storage().NewSeals([]seal.Seal{pr}))

		proposal = pr
	}

	ab := t.newACCEPTBallot(t.localstate, base.Round(0), proposal.Hash(), valuehash.RandomSHA256())

	bc := NewBallotChecker(ab, t.localstate, t.suf)

	err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposal,
	}).Check()
	t.Contains(err.Error(), "publickey not matched")
}

func (t *testBallotChecker) TestCheckWrongHeightProposal() {
	var proposal ballot.Proposal
	{
		pr := ballot.NewProposalV0(
			t.remoteState.Node().Address(),
			t.remoteState.LastBlock().Height()+100, // wrong height
			base.Round(0),
			nil, nil,
		)
		_ = pr.Sign(t.remoteState.Node().Privatekey(), t.remoteState.Policy().NetworkID())
		t.NoError(t.localstate.Storage().NewSeals([]seal.Seal{pr}))

		proposal = pr
	}

	ab := t.newACCEPTBallot(t.localstate, base.Round(0), proposal.Hash(), valuehash.RandomSHA256())

	bc := NewBallotChecker(ab, t.localstate, t.suf)

	err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposal,
	}).Check()
	t.Contains(err.Error(), "proposal in ACCEPTBallot is invalid; different height")
}

func (t *testBallotChecker) TestCheckWrongRoundProposal() {
	var proposal ballot.Proposal
	{
		pr := ballot.NewProposalV0(
			t.remoteState.Node().Address(),
			t.localstate.LastBlock().Height()+1,
			base.Round(33), // wrong round
			nil, nil,
		)
		_ = pr.Sign(t.remoteState.Node().Privatekey(), t.localstate.Policy().NetworkID())
		t.NoError(t.localstate.Storage().NewSeals([]seal.Seal{pr}))

		proposal = pr
	}

	ab := t.newACCEPTBallot(t.localstate, base.Round(0), proposal.Hash(), valuehash.RandomSHA256())

	bc := NewBallotChecker(ab, t.localstate, t.suf)

	err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposal,
	}).Check()
	t.Contains(err.Error(), "proposal in ACCEPTBallot is invalid; different round")
}

func TestBallotChecker(t *testing.T) {
	suite.Run(t, new(testBallotChecker))
}
