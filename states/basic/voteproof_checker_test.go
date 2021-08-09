package basicstates

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
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
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)

	ab := t.NewACCEPTBallot(t.local, ivp.Round(), pr.Hash(), valuehash.RandomSHA256(), ivp)
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
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)
	t.NoError(t.local.Database().NewProposal(pr))

	ab := t.NewACCEPTBallot(t.local, ivp.Round(), pr.Hash(), valuehash.RandomSHA256(), ivp)
	avp, err := t.NewVoteproof(base.StageACCEPT, ab.Fact(), t.local, t.remote)
	t.NoError(err)

	vc := NewVoteproofChecker(t.local.Database(), t.suf, t.local.Nodes(), nil, avp)

	keep, err := vc.CheckACCEPTVoteproofProposal()
	t.True(keep)
	t.NoError(err)
}

func (t *testVoteproofChecker) TestACCEPTVoteproofProposalFoundInRemote() {
	nch := t.remote.Channel().(*channetwork.Channel)
	nch.SetGetSealHandler(func(hs []valuehash.Hash) ([]seal.Seal, error) {
		var seals []seal.Seal
		for _, h := range hs {
			sl, found, err := t.remote.Database().Seal(h)
			if !found {
				break
			} else if err != nil {
				return nil, err
			}

			seals = append(seals, sl)
		}

		return seals, nil
	})

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	initFact := ib.INITFactV0

	ivp, err := t.NewVoteproof(base.StageINIT, initFact, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, initFact.Round(), nil, ivp)
	t.NoError(t.remote.Database().NewProposal(pr))

	ab := t.NewACCEPTBallot(t.local, ivp.Round(), pr.Hash(), valuehash.RandomSHA256(), ivp)
	avp, err := t.NewVoteproof(base.StageACCEPT, ab.Fact(), t.local, t.remote)
	t.NoError(err)

	vc := NewVoteproofChecker(t.local.Database(), t.suf, t.local.Nodes(), nil, avp)

	keep, err := vc.CheckACCEPTVoteproofProposal()
	t.True(keep)
	t.NoError(err)

	npr, found, err := t.local.Database().Proposal(pr.Height(), pr.Round(), pr.Node())
	t.True(found)
	t.NoError(err)
	t.NoError(npr.IsValid(t.local.Policy().NetworkID()))
	t.True(pr.Hash().Equal(npr.Hash()))
}

func TestVoteproofChecker(t *testing.T) {
	suite.Run(t, new(testVoteproofChecker))
}
