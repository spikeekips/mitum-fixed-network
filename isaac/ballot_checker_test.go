package isaac

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	channetwork "github.com/spikeekips/mitum/network/gochan"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testBallotChecker struct {
	BaseTest
	local  *Local
	remote *Local
	suf    base.Suffrage
}

func (t *testBallotChecker) SetupTest() {
	t.BaseTest.SetupTest()

	ls := t.Locals(2)

	t.local, t.remote = ls[0], ls[1]

	t.suf = t.Suffrage(t.remote, t.local)
}

func (t *testBallotChecker) TestIsFromLocal() {
	t.True(t.suf.IsInside(t.local.Node().Address()))

	{ // from local
		ib := t.NewINITBallot(t.local, base.Round(0), nil)
		bc := NewBallotChecker(ib, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))

		var passed bool
		err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.IsFromLocal,
			func() (bool, error) {
				passed = true

				return true, nil
			},
		}).Check()
		t.NoError(err)
		t.False(passed)
	}

	{ // from remote
		ib := t.NewINITBallot(t.remote, base.Round(0), nil)

		bc := NewBallotChecker(ib, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))

		var passed bool
		err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.IsFromLocal,
			func() (bool, error) {
				passed = true

				return true, nil
			},
		}).Check()
		t.NoError(err)
		t.True(passed)
	}
}

func (t *testBallotChecker) TestInTimespan() {
	t.True(t.suf.IsInside(t.local.Node().Address()))

	span := t.local.Policy().TimespanValidBallot()

	{ // too new
		ib := t.NewINITBallot(t.remote, base.Round(0), nil)
		err := ib.SignWithTime(t.remote.Node().Privatekey(), t.local.Policy().NetworkID(), localtime.UTCNow().Add(span+time.Second*10))
		t.NoError(err)

		bc := NewBallotChecker(ib, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))
		err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.InTimespan,
		}).Check()
		t.Contains(err.Error(), "too new ballot")
	}

	{ // too old
		ib := t.NewINITBallot(t.local, base.Round(0), nil)
		err := ib.SignWithTime(t.local.Node().Privatekey(), t.local.Policy().NetworkID(), localtime.UTCNow().Add((span+time.Second*10)*-1))
		t.NoError(err)

		bc := NewBallotChecker(ib, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))
		err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.InTimespan,
		}).Check()
		t.Contains(err.Error(), "too old ballot")
	}
}

func (t *testBallotChecker) TestNew() {
	t.True(t.suf.IsInside(t.local.Node().Address()))

	ib := t.NewINITBallot(t.remote, base.Round(0), nil)

	bc := NewBallotChecker(ib, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))
	err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.InSuffrage,
	}).Check()
	t.NoError(err)
}

func (t *testBallotChecker) TestIsInSuffrage() {
	{ // from local
		t.True(t.suf.IsInside(t.local.Node().Address()))

		ib := t.NewINITBallot(t.remote, base.Round(0), nil)

		bc := NewBallotChecker(ib, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))

		var finished bool
		err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.InSuffrage,
			func() (bool, error) {
				finished = true

				return true, nil
			},
		}).Check()
		t.NoError(err)

		t.True(finished)
	}

	{ // from unknown
		unknown := t.Locals(1)[0]
		t.False(t.suf.IsInside(unknown.Node().Address()))

		ib := t.NewINITBallot(unknown, base.Round(0), nil)

		bc := NewBallotChecker(ib, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))

		var finished bool
		err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.InSuffrage,
			func() (bool, error) {
				finished = true

				return true, nil
			},
		}).Check()
		t.NoError(err)

		t.False(finished)
	}
}

func (t *testBallotChecker) TestCheckWithLastVoteproof() {
	avp := t.local.Database().LastVoteproof(base.StageACCEPT)
	t.NotNil(avp)

	{ // same height and next round
		ibf := t.NewINITBallotFact(t.remote, base.Round(1))
		vp, _ := t.NewVoteproof(base.StageINIT, ibf, t.local, t.remote)

		ib := t.NewINITBallot(t.remote, vp.Round()+1, vp)

		bc := NewBallotChecker(ib, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))

		var finished bool
		err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.CheckWithLastVoteproof,
			func() (bool, error) {
				finished = true

				return true, nil
			},
		}).Check()
		t.NoError(err)

		t.True(finished)
	}

	{ // lower Height
		lastManifest := t.LastManifest(t.local.Database())

		ib := ballot.NewINITV0(
			t.local.Node().Address(),
			lastManifest.Height(),
			base.Round(0),
			lastManifest.Hash(),
			avp,
			avp,
		)

		t.NoError(ib.Sign(t.local.Node().Privatekey(), t.local.Policy().NetworkID()))

		bc := NewBallotChecker(ib, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))

		var finished bool
		err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
			bc.CheckWithLastVoteproof,
			func() (bool, error) {
				finished = true

				return true, nil
			},
		}).Check()
		t.NoError(err)

		t.False(finished)
	}
}

func (t *testBallotChecker) TestCheckProposalInACCEPTBallotWithKnownProposal() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	ivp, err := t.NewVoteproof(base.StageINIT, ib.INITFactV0, t.local, t.remote)
	t.NoError(err)

	pr := t.NewProposal(t.remote, ivp.Round(), nil, ivp)

	// NOTE save the remote proposal in local
	t.NoError(t.local.Database().NewProposal(pr))

	upr, found, err := t.local.Database().Seal(pr.Hash())
	t.NoError(err)
	t.True(found)
	t.NotNil(upr)

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Hash(), valuehash.RandomSHA256())

	ab := t.NewACCEPTBallot(t.remote, ivp.Round(), newblock.Proposal(), newblock.Hash(), nil)

	bc := NewBallotChecker(ab, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))

	var finished bool
	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposalInACCEPTBallot,
		func() (bool, error) {
			finished = true

			return true, nil
		},
	}).Check()
	t.NoError(err)

	t.True(finished)
}

func (t *testBallotChecker) TestCheckProposalInACCEPTBallotWithUnknownProposalAndFoundInProposer() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	ivp, err := t.NewVoteproof(base.StageINIT, ib.INITFactV0, t.local, t.remote)
	t.NoError(err)

	// NOTE remote is proposer
	pr := t.NewProposal(t.remote, ivp.Round(), nil, ivp)

	// NOTE remote knows proposal
	t.NoError(t.remote.Database().NewProposal(pr))

	_, ch, found := t.remote.Nodes().Node(t.remote.Node().Address())
	t.True(found)
	t.NotNil(ch)
	nch := ch.(*channetwork.Channel)
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

	upr, found, err := t.local.Database().Seal(pr.Hash())
	t.NoError(err)
	t.False(found)
	t.Nil(upr)

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Hash(), valuehash.RandomSHA256())

	ab := t.NewACCEPTBallot(t.remote, ivp.Round(), newblock.Proposal(), newblock.Hash(), nil)

	bc := NewBallotChecker(ab, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))

	var finished bool
	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposalInACCEPTBallot,
		func() (bool, error) {
			finished = true

			return true, nil
		},
	}).Check()
	t.NoError(err)

	t.True(finished)
}

func (t *testBallotChecker) TestCheckProposalInACCEPTBallotWithUnknownProposalButNotFound() {
	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	ivp, err := t.NewVoteproof(base.StageINIT, ib.INITFactV0, t.local, t.remote)
	t.NoError(err)

	// NOTE remote is proposer
	pr := t.NewProposal(t.remote, ivp.Round(), nil, ivp)

	_, ch, found := t.remote.Nodes().Node(t.remote.Node().Address())
	t.True(found)
	t.NotNil(ch)
	nch := ch.(*channetwork.Channel)
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

	upr, found, err := t.local.Database().Seal(pr.Hash())
	t.NoError(err)
	t.False(found)
	t.Nil(upr)

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Hash(), valuehash.RandomSHA256())

	ab := t.NewACCEPTBallot(t.remote, ivp.Round(), newblock.Proposal(), newblock.Hash(), nil)

	bc := NewBallotChecker(ab, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))

	var finished bool
	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposalInACCEPTBallot,
		func() (bool, error) {
			finished = true

			return true, nil
		},
	}).Check()
	t.Contains(err.Error(), "failed to request proposal")

	t.False(finished)
}

func (t *testBallotChecker) TestCheckProposalInACCEPTBallotWithUnknownProposalButFoundInOther() {
	other := t.Locals(1)[0]

	all := []*Local{t.local, t.remote, other}
	for _, l := range all {
		for _, r := range all {
			if err := l.Nodes().Add(r.Node(), r.Channel()); err != nil {
				continue
			}

			if err := l.Nodes().Add(r.Node(), r.Channel()); err != nil {
				if errors.Is(err, util.FoundError) {
					continue
				}
				panic(err)
			}
		}
	}

	suffrage := t.Suffrage(t.remote, t.local, other)

	ib := t.NewINITBallot(t.local, base.Round(0), nil)
	ivp, err := t.NewVoteproof(base.StageINIT, ib.INITFactV0, t.local, t.remote)
	t.NoError(err)

	// NOTE remote is proposer
	pr := t.NewProposal(t.remote, ivp.Round(), nil, ivp)

	// NOTE save the other proposal in local
	t.NoError(other.Database().NewProposal(pr))

	rch := t.remote.Channel().(*channetwork.Channel)
	rch.SetGetSealHandler(func(hs []valuehash.Hash) ([]seal.Seal, error) {
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

	och := other.Channel().(*channetwork.Channel)
	och.SetGetSealHandler(func(hs []valuehash.Hash) ([]seal.Seal, error) {
		var seals []seal.Seal
		for _, h := range hs {
			sl, found, err := other.Database().Seal(h)
			if !found {
				break
			} else if err != nil {
				return nil, err
			}

			seals = append(seals, sl)
		}

		return seals, nil
	})

	upr, found, err := t.local.Database().Seal(pr.Hash())
	t.NoError(err)
	t.False(found)
	t.Nil(upr)

	newblock, _ := block.NewTestBlockV0(ivp.Height(), ivp.Round(), pr.Hash(), valuehash.RandomSHA256())

	ab := t.NewACCEPTBallot(t.remote, ivp.Round(), newblock.Proposal(), newblock.Hash(), nil)

	bc := NewBallotChecker(ab, t.local.Database(), t.local.Policy(), suffrage, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))

	var finished bool
	err = util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.CheckProposalInACCEPTBallot,
		func() (bool, error) {
			finished = true

			return true, nil
		},
	}).Check()
	t.NoError(err)

	t.True(finished)
}

func TestBallotChecker(t *testing.T) {
	suite.Run(t, new(testBallotChecker))
}
