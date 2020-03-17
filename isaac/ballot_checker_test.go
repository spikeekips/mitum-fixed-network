package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

type testBallotChecker struct {
	baseTestStateHandler

	suf Suffrage
}

func (t *testBallotChecker) SetupTest() {
	t.baseTestStateHandler.SetupTest()

	t.suf = t.suffrage(t.remoteState, t.localstate)
}

func (t *testBallotChecker) TestNew() {
	t.True(t.suf.IsInside(t.localstate.Node().Address()))

	ib, err := NewINITBallotV0FromLocalstate(t.localstate, Round(0))
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

		ib, err := NewINITBallotV0FromLocalstate(t.localstate, Round(0))
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

		ib, err := NewINITBallotV0FromLocalstate(unknown, Round(0))
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
		ib, err := NewINITBallotV0FromLocalstate(t.localstate, Round(0))
		t.NoError(err)

		ib.INITBallotFactV0.height = ib.INITBallotFactV0.height - 1
		t.NoError(ib.Sign(
			t.localstate.Node().Privatekey(),
			t.localstate.Policy().NetworkID(),
		))

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

		t.False(finished)
	}
}

func TestBallotChecker(t *testing.T) {
	suite.Run(t, new(testBallotChecker))
}
