package isaac

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/node"
	"github.com/stretchr/testify/suite"
)

type testProposalChecker struct {
	BaseTest
	local  *Local
	remote *Local
	suf    base.Suffrage
}

func (t *testProposalChecker) SetupTest() {
	t.BaseTest.SetupTest()

	ls := t.Locals(2)
	t.local, t.remote = ls[0], ls[1]
	t.suf = t.Suffrage(t.remote, t.local)
}

func (t *testProposalChecker) TestIsKnown() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, vp)

	pvc, err := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, nil)
	t.NoError(err)

	{
		keep, err := pvc.IsKnown()
		t.True(keep)
		t.NoError(err)
	}

	{
		// NOTE store proposal
		t.NoError(t.local.Database().NewProposal(pr))
		keep, err := pvc.IsKnown()
		t.False(keep)
		t.Error(err)
		t.True(errors.Is(err, KnownSealError))
	}
}

func (t *testProposalChecker) TestCheckSigning() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	{
		pr := t.NewProposal(t.remote, initFact.Round(), nil, vp)

		pvc, err := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, nil)
		t.NoError(err)
		keep, err := pvc.CheckSigning()
		t.True(keep)
		t.NoError(err)
	}

	{
		ls := t.Locals(1)
		another := ls[0]
		another.SetNode(node.NewLocal(t.local.Node().Address(), another.Node().Privatekey()))

		npr := t.NewProposal(another, initFact.Round(), nil, vp)

		pvc, err := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), npr, nil)
		t.NoError(err)

		// NOTE sign again with another privatekey
		keep, err := pvc.CheckSigning()
		t.False(keep)
		t.Error(err)
		t.Contains(err.Error(), "publickey not matched")
	}
}

func (t *testProposalChecker) TestProposerPointProposal() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, vp)

	pvc, err := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, vp)
	t.NoError(err)
	keep, err := pvc.IsOlder()
	t.True(keep)
	t.NoError(err)
}

func (t *testProposalChecker) TestProposerPointOldProposalHeight() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr, err := ballot.NewProposal(
		ballot.NewProposalFact(
			ivp.Height()-1,
			base.Round(0),
			t.local.Node().Address(),
			nil,
		),
		t.local.Node().Address(),
		nil,
		t.local.Node().Privatekey(), t.local.Policy().NetworkID(),
	)
	t.NoError(err)

	pvc, err := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, ivp)
	t.NoError(err)
	keep, err := pvc.IsOlder()
	t.False(keep)
	t.Error(err)
	t.Contains(err.Error(), "lower proposal height than last voteproof")
}

func (t *testProposalChecker) TestProposerPointOldProposalRound() {
	ib0 := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact0 := ib0.Fact()

	ivp0, err := t.NewVoteproof(base.StageINIT, initFact0, t.local, t.remote)
	t.NoError(err)

	ib1 := t.NewINITBallot(t.local, base.Round(1), ivp0)
	initFact1 := ib1.Fact()

	ivp1, err := t.NewVoteproof(base.StageINIT, initFact1, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, base.Round(0), nil, ivp0)

	pvc, err := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, ivp1)
	t.NoError(err)
	keep, err := pvc.IsOlder()
	t.False(keep)
	t.Error(err)
	t.Contains(err.Error(), "same height, but lower proposal round than last voteproof")
}

func (t *testProposalChecker) TestProposerPointHigherProposalHeight() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr, err := ballot.NewProposal(
		ballot.NewProposalFact(
			ivp.Height()+1,
			base.Round(0),
			t.local.Node().Address(),
			nil,
		),
		t.local.Node().Address(),
		nil,
		t.local.Node().Privatekey(), t.local.Policy().NetworkID(),
	)
	t.NoError(err)

	pvc, err := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, ivp)
	t.NoError(err)

	keep, err := pvc.IsOlder()
	t.True(keep)
	t.NoError(err)

	keep, err = pvc.IsWaiting()
	t.False(keep)
	t.Error(err)
	t.Contains(err.Error(), "proposal height does not match with last voteproof")
}

func (t *testProposalChecker) TestProposerPointHigherProposalRound() {
	ib0 := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact0 := ib0.Fact()

	ivp0, err := t.NewVoteproof(base.StageINIT, initFact0, t.local, t.remote)
	t.NoError(err)

	ib1 := t.NewINITBallot(t.local, base.Round(1), ivp0)
	initFact1 := ib1.Fact()

	ivp1, err := t.NewVoteproof(base.StageINIT, initFact1, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, base.Round(1), nil, ivp1)

	pvc, err := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, ivp0)
	t.NoError(err)
	keep, err := pvc.IsWaiting()
	t.False(keep)
	t.Error(err)
	t.Contains(err.Error(), "proposal round does not match with last voteproof")
}

func TestProposalChecker(t *testing.T) {
	suite.Run(t, new(testProposalChecker))
}
