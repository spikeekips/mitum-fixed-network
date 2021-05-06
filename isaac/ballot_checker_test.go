package isaac

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/localtime"
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

func (t *testBallotChecker) TestInTimespan() {
	t.True(t.suf.IsInside(t.local.Node().Address()))

	span := t.local.Policy().TimespanValidBallot()

	{ // too new
		ib := t.NewINITBallot(t.local, base.Round(0), nil)
		err := ib.SignWithTime(t.local.Node().Privatekey(), t.local.Policy().NetworkID(), localtime.UTCNow().Add(span+time.Second*10))
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

	ib := t.NewINITBallot(t.local, base.Round(0), nil)

	bc := NewBallotChecker(ib, t.local.Database(), t.local.Policy(), t.suf, t.local.Nodes(), t.local.Database().LastVoteproof(base.StageINIT))
	err := util.NewChecker("test-ballot-checker", []util.CheckerFunc{
		bc.InSuffrage,
	}).Check()
	t.NoError(err)
}

func (t *testBallotChecker) TestIsInSuffrage() {
	{ // from local
		t.True(t.suf.IsInside(t.local.Node().Address()))

		ib := t.NewINITBallot(t.local, base.Round(0), nil)

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
		ibf := t.NewINITBallotFact(t.local, base.Round(1))
		vp, _ := t.NewVoteproof(base.StageINIT, ibf, t.local, t.remote)

		ib := t.NewINITBallot(t.local, vp.Round()+1, vp)

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

		ib := ballot.NewINITBallotV0(
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

func TestBallotChecker(t *testing.T) {
	suite.Run(t, new(testBallotChecker))
}
