package ballot

import (
	"errors"
	"reflect"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/stretchr/testify/suite"
)

type baseTest struct {
	suite.Suite
	pk        key.Privatekey
	networkID base.NetworkID
}

func (t *baseTest) SetupSuite() {
	t.pk = key.NewBasePrivatekey()
	t.networkID = base.NetworkID([]byte("showme"))
}

func (t *baseTest) compareFact(a, b base.BallotFact) {
	if a == nil {
		if b != nil {
			t.True(false, "b is not nil")
			return
		}

		return
	}

	t.True(a.Hint().Equal(b.Hint()))
	t.True(a.Hash().Equal(b.Hash()))
	t.Equal(a.Stage(), b.Stage())
	t.Equal(a.Height(), b.Height())
	t.Equal(a.Round(), b.Round())

	switch xa := a.(type) {
	case INITFact:
		xb := b.(INITFact)
		t.True(xa.PreviousBlock().Equal(xb.PreviousBlock()))
	case ProposalFact:
		xb := b.(ProposalFact)

		t.True(xa.proposer.Equal(xb.proposer))

		t.Equal(len(xa.Seals()), len(xb.Seals()))
		as := xa.Seals()
		bs := xb.Seals()
		for i := range as {
			t.True(as[i].Equal(bs[i]))
		}
	case ACCEPTFact:
		xb := b.(ACCEPTFact)
		t.True(xa.Proposal().Equal(xb.Proposal()))
		t.True(xa.NewBlock().Equal(xb.NewBlock()))
	}
}

func (t *baseTest) compareBallot(a, b base.Ballot) {
	if a == nil {
		if b != nil {
			t.True(false, "b is not nil")
			return
		}

		return
	}

	t.True(a.Hash().Equal(b.Hash()))
	t.True(a.Hint().Equal(b.Hint()))
	t.True(a.BodyHash().Equal(b.BodyHash()))
	t.True(a.Signer().Equal(b.Signer()))
	t.True(a.Signature().Equal(b.Signature()))
	t.True(localtime.Equal(a.SignedAt(), b.SignedAt()))

	t.compareFact(a.RawFact(), b.RawFact())

	t.True(a.FactSign().Signer().Equal(b.FactSign().Signer()))
	t.Equal(a.FactSign().Signature(), b.FactSign().Signature())
	t.True(localtime.Equal(a.FactSign().SignedAt(), b.FactSign().SignedAt()))
}

type testSuite struct {
	baseTest
	newFact   func(base.Height, base.Round) base.BallotFact
	newBallot func(
		base.BallotFact,
		base.Address,
		base.Voteproof,
		base.Voteproof,
	) (base.Ballot, error)
}

func (t *testSuite) TestNoneEmptyACCEPTVoteproof() {
	if t.newFact == nil {
		return
	}

	height := base.Height(3)
	round := base.Round(0)

	fact := t.newFact(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round,
		base.StageINIT,
		base.VoteResultMajority,
	)

	avp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := t.newBallot(fact, base.RandomStringAddress(), bvp, avp)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.NotNil(err)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "not empty accept voteproof with base voteproof")
}

func (t *testSuite) TestWrongStageBaseVoteproof() {
	if t.newFact == nil {
		return
	}

	height := base.Height(3)
	round := base.Round(0)

	fact := t.newFact(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round,
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := t.newBallot(fact, base.RandomStringAddress(), bvp, nil)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.NotNil(err)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "invalid base voteproof stage")
}

func (t *testSuite) TestWrongHeightBaseVoteproof() {
	if t.newFact == nil {
		return
	}

	height := base.Height(3)
	round := base.Round(0)

	fact := t.newFact(height, round)

	bvp := base.NewDummyVoteproof(
		height+1,
		round,
		base.StageINIT,
		base.VoteResultMajority,
	)

	sl, err := t.newBallot(fact, base.RandomStringAddress(), bvp, nil)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.NotNil(err)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong height of base voteproof")
}

func (t *testSuite) TestWrongRoundBaseVoteproof() {
	if t.newFact == nil {
		return
	}

	height := base.Height(3)
	round := base.Round(0)

	fact := t.newFact(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round+1,
		base.StageINIT,
		base.VoteResultMajority,
	)

	sl, err := t.newBallot(fact, base.RandomStringAddress(), bvp, nil)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.NotNil(err)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong round of base voteproof")
}

func (t *testSuite) TestWrongResultBaseVoteproof() {
	if t.newFact == nil {
		return
	}

	height := base.Height(3)
	round := base.Round(0)

	fact := t.newFact(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round+1,
		base.StageINIT,
		base.VoteResultDraw,
	)

	sl, err := t.newBallot(fact, base.RandomStringAddress(), bvp, nil)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.NotNil(err)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "not majority result of base voteproof")
}

func (t *testSuite) testSignWithFact(sl base.SignWithFacter, n base.Address) {
	osl := reflect.ValueOf(sl).Elem().Interface().(base.Ballot)

	// sign again
	t.NoError(sl.Sign(t.pk, t.networkID))

	asl := reflect.ValueOf(sl).Elem().Interface().(base.Ballot)

	t.compareFact(osl.RawFact(), asl.RawFact())

	t.True(osl.SignedAt().Before(asl.SignedAt()))
	t.True(osl.FactSign().Node().Equal(asl.FactSign().Node()))
	t.Equal(osl.FactSign().SignedAt(), asl.FactSign().SignedAt())
	t.Equal(osl.FactSign().Signature(), asl.FactSign().Signature())

	// sign again
	t.NoError(sl.SignWithFact(osl.FactSign().Node(), t.pk, t.networkID))

	csl := reflect.ValueOf(sl).Elem().Interface().(base.Ballot)

	t.compareFact(osl.RawFact(), csl.RawFact())

	t.True(asl.SignedAt().Before(csl.SignedAt()))
	t.True(asl.FactSign().Node().Equal(csl.FactSign().Node()))
	t.True(asl.FactSign().SignedAt().Before(csl.FactSign().SignedAt()))
	t.NotEqual(asl.FactSign().Signature(), csl.FactSign().Signature())
}
