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

	ib := t.newINITBallot(t.localstate, base.Round(0), nil)

	bc, err := NewBallotChecker(ib, t.localstate, t.suf)
	t.NoError(err)
	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckIsInSuffrage,
	}).Check()
	t.NoError(err)
}

func (t *testBallotChecker) TestIsInSuffrage() {
	{ // from localstate
		t.True(t.suf.IsInside(t.localstate.Node().Address()))

		ib := t.newINITBallot(t.localstate, base.Round(0), nil)

		bc, err := NewBallotChecker(ib, t.localstate, t.suf)
		t.NoError(err)

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
		unknown := t.localstates(1)[0]
		t.False(t.suf.IsInside(unknown.Node().Address()))

		ib := t.newINITBallot(unknown, base.Round(0), nil)

		bc, err := NewBallotChecker(ib, t.localstate, t.suf)
		t.NoError(err)

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
	var avp base.Voteproof

	avp, err := t.localstate.Storage().LastVoteproof(base.StageACCEPT)
	t.NoError(err)

	{ // same height and next round
		ib := t.newINITBallot(t.localstate, avp.Round()+1, t.lastINITVoteproof(t.localstate))

		bc, err := NewBallotChecker(ib, t.localstate, t.suf)
		t.NoError(err)

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
		lastManifest := t.lastManifest(t.localstate.Storage())

		ib := ballot.NewINITBallotV0(
			t.localstate.Node().Address(),
			lastManifest.Height(),
			base.Round(0),
			lastManifest.Hash(),
			avp,
		)

		t.NoError(ib.Sign(t.localstate.Node().Privatekey(), t.localstate.Policy().NetworkID()))

		bc, err := NewBallotChecker(ib, t.localstate, t.suf)
		t.NoError(err)

		var finished bool
		err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
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
			t.lastManifest(t.localstate.Storage()).Height()+1,
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

	bc, err := NewBallotChecker(ab, t.localstate, t.suf)
	t.NoError(err)

	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposal,
	}).Check()
	t.Contains(err.Error(), "publickey not matched")
}

func (t *testBallotChecker) TestCheckWrongHeightProposal() {
	var proposal ballot.Proposal
	{
		pr := ballot.NewProposalV0(
			t.remoteState.Node().Address(),
			t.lastManifest(t.remoteState.Storage()).Height()+100, // wrong height
			base.Round(0),
			nil, nil,
		)
		_ = pr.Sign(t.remoteState.Node().Privatekey(), t.remoteState.Policy().NetworkID())
		t.NoError(t.localstate.Storage().NewSeals([]seal.Seal{pr}))

		proposal = pr
	}

	ab := t.newACCEPTBallot(t.localstate, base.Round(0), proposal.Hash(), valuehash.RandomSHA256())

	bc, err := NewBallotChecker(ab, t.localstate, t.suf)
	t.NoError(err)

	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposal,
	}).Check()
	t.Contains(err.Error(), "proposal in ACCEPTBallot is invalid; different height")
}

func (t *testBallotChecker) TestCheckWrongRoundProposal() {
	var proposal ballot.Proposal
	{
		pr := ballot.NewProposalV0(
			t.remoteState.Node().Address(),
			t.lastManifest(t.localstate.Storage()).Height()+1,
			base.Round(33), // wrong round
			nil, nil,
		)
		_ = pr.Sign(t.remoteState.Node().Privatekey(), t.localstate.Policy().NetworkID())
		t.NoError(t.localstate.Storage().NewSeals([]seal.Seal{pr}))

		proposal = pr
	}

	ab := t.newACCEPTBallot(t.localstate, base.Round(0), proposal.Hash(), valuehash.RandomSHA256())

	bc, err := NewBallotChecker(ab, t.localstate, t.suf)
	t.NoError(err)

	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposal,
	}).Check()
	t.Contains(err.Error(), "proposal in ACCEPTBallot is invalid; different round")
}

func TestBallotChecker(t *testing.T) {
	suite.Run(t, new(testBallotChecker))
}
