package basicstates

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testVoteproofChecker struct {
	baseTestState

	suf base.Suffrage
}

func (t *testVoteproofChecker) SetupTest() {
	t.baseTestState.SetupTest()

	t.suf = t.Suffrage(t.remote, t.local)
}

func (t *testVoteproofChecker) TestACCEPTVoteproofProposalNotFound() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	ab := t.NewACCEPTBallot(t.local, ivp.Round(), pr.Fact().Hash(), valuehash.RandomSHA256(), ivp)
	avp, err := t.NewVoteproof(base.StageACCEPT, ab.Fact(), t.local, t.remote)
	t.NoError(err)

	vc := NewVoteproofChecker(t.local.Database(), t.suf, t.local.Nodes(), nil, avp)

	keep, err := vc.CheckACCEPTVoteproofProposal()
	t.False(keep)
	t.Error(err)
	t.Contains(err.Error(), "failed to find proposal from accept voteproof")
}

func (t *testVoteproofChecker) TestACCEPTVoteproofProposalFoundInLocal() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)
	t.NoError(t.local.Database().NewProposal(pr))

	ab := t.NewACCEPTBallot(t.local, ivp.Round(), pr.Fact().Hash(), valuehash.RandomSHA256(), ivp)
	avp, err := t.NewVoteproof(base.StageACCEPT, ab.Fact(), t.local, t.remote)
	t.NoError(err)

	vc := NewVoteproofChecker(t.local.Database(), t.suf, t.local.Nodes(), nil, avp)

	keep, err := vc.CheckACCEPTVoteproofProposal()
	t.True(keep)
	t.NoError(err)
}

func (t *testVoteproofChecker) TestACCEPTVoteproofProposalFoundInRemote() {
	nch := t.remote.Channel().(*channetwork.Channel)
	nch.SetGetProposalHandler(func(h valuehash.Hash) (base.Proposal, error) {
		i, _, err := t.remote.Database().Proposal(h)

		return i, err
	})

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.Fact()

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)
	t.NoError(ivp.IsValid(t.local.Policy().NetworkID()))

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)
	t.NoError(pr.IsValid(t.local.Policy().NetworkID()))
	t.NoError(t.remote.Database().NewProposal(pr))

	ab := t.NewACCEPTBallot(t.local, ivp.Round(), pr.Fact().Hash(), valuehash.RandomSHA256(), ivp)
	avp, err := t.NewVoteproof(base.StageACCEPT, ab.Fact(), t.local, t.remote)
	t.NoError(err)

	vc := NewVoteproofChecker(t.local.Database(), t.suf, t.local.Nodes(), nil, avp)

	keep, err := vc.CheckACCEPTVoteproofProposal()
	t.True(keep)
	t.NoError(err)

	npr, found, err := t.local.Database().ProposalByPoint(pr.Fact().Height(), pr.Fact().Round(), pr.FactSign().Node())
	t.True(found)
	t.NoError(err)
	t.NoError(npr.IsValid(t.local.Policy().NetworkID()))
	t.True(pr.Hash().Equal(npr.Hash()))
}

func TestVoteproofChecker(t *testing.T) {
	suite.Run(t, new(testVoteproofChecker))
}
