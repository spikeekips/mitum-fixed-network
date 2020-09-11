package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testBallotChecker struct {
	baseTestStateHandler

	suf base.Suffrage

	local  *Localstate
	remote *Localstate
}

func (t *testBallotChecker) SetupTest() {
	t.baseTestStateHandler.SetupTest()

	ls := t.localstates(2)

	t.local, t.remote = ls[0], ls[1]

	t.suf = t.suffrage(t.remote, t.local)
}

func (t *testBallotChecker) TestNew() {
	t.True(t.suf.IsInside(t.local.Node().Address()))

	ib := t.newINITBallot(t.local, base.Round(0), nil)

	bc, err := NewBallotChecker(ib, t.local, t.suf)
	t.NoError(err)
	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckIsInSuffrage,
	}).Check()
	t.NoError(err)
}

func (t *testBallotChecker) TestIsInSuffrage() {
	{ // from local
		t.True(t.suf.IsInside(t.local.Node().Address()))

		ib := t.newINITBallot(t.local, base.Round(0), nil)

		bc, err := NewBallotChecker(ib, t.local, t.suf)
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

		bc, err := NewBallotChecker(ib, t.local, t.suf)
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

	avp, found, err := t.local.BlockFS().LastVoteproof(base.StageACCEPT)
	t.NoError(err)
	t.True(found)

	{ // same height and next round
		ibf := t.newINITBallotFact(t.local, base.Round(1))
		vp, _ := t.newVoteproof(base.StageINIT, ibf, t.local, t.remote)

		ib := t.newINITBallot(t.local, vp.Round()+1, vp)

		bc, err := NewBallotChecker(ib, t.local, t.suf)
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
		lastManifest := t.lastManifest(t.local.Storage())

		ib := ballot.NewINITBallotV0(
			t.local.Node().Address(),
			lastManifest.Height(),
			base.Round(0),
			lastManifest.Hash(),
			avp,
		)

		t.NoError(ib.Sign(t.local.Node().Privatekey(), t.local.Policy().NetworkID()))

		bc, err := NewBallotChecker(ib, t.local, t.suf)
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
			t.local.Node().Address(),
			t.lastManifest(t.local.Storage()).Height()+1,
			base.Round(0),
			nil,
		)

		// signed by unknown node
		pk, _ := key.NewBTCPrivatekey()
		_ = pr.Sign(pk, t.local.Policy().NetworkID())
		t.NoError(t.local.Storage().NewSeals([]seal.Seal{pr}))

		proposal = pr
	}

	ab := t.newACCEPTBallot(t.local, base.Round(0), proposal.Hash(), valuehash.RandomSHA256())

	bc, err := NewBallotChecker(ab, t.local, t.suf)
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
			t.remote.Node().Address(),
			t.lastManifest(t.remote.Storage()).Height()+100, // wrong height
			base.Round(0),
			nil,
		)
		_ = pr.Sign(t.remote.Node().Privatekey(), t.remote.Policy().NetworkID())
		t.NoError(t.local.Storage().NewSeals([]seal.Seal{pr}))

		proposal = pr
	}

	ab := t.newACCEPTBallot(t.local, base.Round(0), proposal.Hash(), valuehash.RandomSHA256())

	bc, err := NewBallotChecker(ab, t.local, t.suf)
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
			t.remote.Node().Address(),
			t.lastManifest(t.local.Storage()).Height()+1,
			base.Round(33), // wrong round
			nil,
		)
		_ = pr.Sign(t.remote.Node().Privatekey(), t.local.Policy().NetworkID())
		t.NoError(t.local.Storage().NewSeals([]seal.Seal{pr}))

		proposal = pr
	}

	ab := t.newACCEPTBallot(t.local, base.Round(0), proposal.Hash(), valuehash.RandomSHA256())

	bc, err := NewBallotChecker(ab, t.local, t.suf)
	t.NoError(err)

	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposal,
	}).Check()
	t.Contains(err.Error(), "proposal in ACCEPTBallot is invalid; different round")
}

func TestBallotChecker(t *testing.T) {
	suite.Run(t, new(testBallotChecker))
}
