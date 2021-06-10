package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
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
	initFact := ib.INITFactV0

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, vp)

	pvc := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, nil)

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
		t.True(xerrors.Is(err, KnownSealError))
	}
}

func (t *testProposalChecker) TestCheckSigning() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, vp)

	{
		pvc := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, nil)
		keep, err := pvc.CheckSigning()
		t.True(keep)
		t.NoError(err)
	}

	{
		npr := pr.(ballot.ProposalV0)

		t.NoError(SignSeal(&npr, t.local))

		pvc := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), npr, nil)

		// NOTE sign again with another privatekey
		keep, err := pvc.CheckSigning()
		t.False(keep)
		t.Error(err)
		t.Contains(err.Error(), "publickey not matched")
	}
}

func (t *testProposalChecker) TestPropserPointProposal() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	vp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, vp)

	pvc := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, vp)
	keep, err := pvc.IsOlder()
	t.True(keep)
	t.NoError(err)
}

func (t *testProposalChecker) TestPropserPointOldProposalHeight() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := ballot.NewProposalV0(
		t.local.Node().Address(),
		ivp.Height()-1,
		base.Round(0),
		nil,
		nil,
	)

	pvc := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, ivp)
	keep, err := pvc.IsOlder()
	t.False(keep)
	t.Error(err)
	t.Contains(err.Error(), "lower proposal height than last voteproof")
}

func (t *testProposalChecker) TestPropserPointOldProposalRound() {
	ib0 := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact0 := ib0.INITFactV0

	ivp0, err := t.NewVoteproof(base.StageINIT, initFact0, t.local, t.remote)
	t.NoError(err)

	ib1 := t.NewINITBallot(t.local, base.Round(1), ivp0)
	initFact1 := ib1.INITFactV0

	ivp1, err := t.NewVoteproof(base.StageINIT, initFact1, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, base.Round(0), nil, ivp0)

	pvc := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, ivp1)
	keep, err := pvc.IsOlder()
	t.False(keep)
	t.Error(err)
	t.Contains(err.Error(), "same height, but lower proposal round than last voteproof")
}

func (t *testProposalChecker) TestPropserPointHigherProposalHeight() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := ballot.NewProposalV0(
		t.local.Node().Address(),
		ivp.Height()+1,
		base.Round(0),
		nil,
		nil,
	)

	pvc := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, ivp)

	keep, err := pvc.IsOlder()
	t.True(keep)
	t.NoError(err)

	keep, err = pvc.IsWaiting()
	t.False(keep)
	t.Error(err)
	t.Contains(err.Error(), "proposal height does not match with last voteproof")
}

func (t *testProposalChecker) TestPropserPointHigherProposalRound() {
	ib0 := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact0 := ib0.INITFactV0

	ivp0, err := t.NewVoteproof(base.StageINIT, initFact0, t.local, t.remote)
	t.NoError(err)

	ib1 := t.NewINITBallot(t.local, base.Round(1), ivp0)
	initFact1 := ib1.INITFactV0

	ivp1, err := t.NewVoteproof(base.StageINIT, initFact1, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, base.Round(1), nil, ivp1)

	pvc := NewProposalValidationChecker(t.local.Database(), t.suf, t.local.Nodes(), pr, ivp0)
	keep, err := pvc.IsWaiting()
	t.False(keep)
	t.Error(err)
	t.Contains(err.Error(), "proposal round does not match with last voteproof")
}

func TestProposalChecker(t *testing.T) {
	suite.Run(t, new(testProposalChecker))
}
