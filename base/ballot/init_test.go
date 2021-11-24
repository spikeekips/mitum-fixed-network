package ballot

import (
	"errors"
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testINITFact struct {
	suite.Suite
}

func (t *testINITFact) TestNew() {
	height := base.Height(3)
	round := base.Round(33)
	previousBlock := valuehash.RandomSHA256()

	fact := NewINITFact(
		height,
		round,
		previousBlock,
	)
	t.NoError(fact.IsValid(nil))

	_, ok := (interface{})(fact).(base.INITBallotFact)
	t.True(ok)

	t.Equal(height, fact.Height())
	t.Equal(round, fact.Round())
	t.True(previousBlock.Equal(fact.PreviousBlock()))
}

func (t *testINITFact) TestEmptyPreviousBlock() {
	fact := NewINITFact(
		base.Height(3),
		base.Round(33),
		nil,
	)

	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testINITFact) TestHashNotMatched() {
	fact := NewINITFact(
		base.Height(3),
		base.Round(33),
		valuehash.RandomSHA256(),
	)
	t.NoError(fact.IsValid(nil))

	fact.h = valuehash.RandomSHA256()
	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "hash does not match")
}

func TestINITFact(t *testing.T) {
	suite.Run(t, new(testINITFact))
}

type testINIT struct {
	testSuite
}

func (t *testINIT) TestSign() {
	height := base.Height(3)
	round := base.Round(0)
	previousBlock := valuehash.RandomSHA256()

	fact := NewINITFact(
		height,
		round,
		previousBlock,
	)

	bavp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bavp, nil, t.pk, t.networkID)
	t.NoError(err)

	t.Implements((*seal.Seal)(nil), sl)

	_ = (interface{})(sl).(base.Ballot)
	t.Implements((*base.Ballot)(nil), sl)

	_ = (interface{})(sl).(base.INITBallot)
	t.Implements((*base.INITBallot)(nil), sl)

	t.NoError(sl.IsValid(t.networkID))

	t.compareFact(fact, sl.Fact())
	t.Equal(sl.Fact().Stage(), base.StageINIT)
	t.True(sl.Hint().Equal(INITHint))

	t.testSignWithFact(&sl, sl.FactSign().Node())
}

func (t *testINIT) factSign(height base.Height, round base.Round) INITFact {
	return NewINITFact(
		height,
		round,
		valuehash.RandomSHA256(),
	)
}

func (t *testINIT) TestACCEPTBaseVoteproofNotNilACCEPTVoteproof() {
	height := base.Height(3)
	round := base.Round(0)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, bvp, t.pk, t.networkID) // same accept voteproof with base voteproof
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "not empty accept voteproof with accept base voteproof")
}

func (t *testINIT) TestACCEPTBaseVoteproofWrongVoteproofHeight() {
	height := base.Height(3)
	round := base.Round(0)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, nil, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong height of init ballot")
}

func (t *testINIT) TestACCEPTBaseVoteproofWrongVoteproofRound() {
	height := base.Height(3)
	round := base.Round(33)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, nil, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong round of init ballot")
}

func (t *testINIT) TestDRAWACCEPTBaseVoteproofEmptyACCEPTVoteproof() {
	height := base.Height(3)
	round := base.Round(0)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultDraw,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, nil, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "empty accept voteproof with draw accept base voteproof")
}

func (t *testINIT) TestDRAWACCEPTBaseVoteproofWrongVoteproofHeight() {
	height := base.Height(3)
	round := base.Round(0)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultDraw,
	)

	avp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, avp, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong height of init ballot")
}

func (t *testINIT) TestDRAWACCEPTBaseVoteproofWrongVoteproofRound() {
	height := base.Height(3)
	round := base.Round(1)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round,
		base.StageACCEPT,
		base.VoteResultDraw,
	)

	avp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, avp, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong round of init ballot")
}

func (t *testINIT) TestDRAWACCEPTBaseVoteproofWrongACCEPTVoteproofStage() {
	height := base.Height(3)
	round := base.Round(33)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round-1,
		base.StageACCEPT,
		base.VoteResultDraw,
	)

	avp := base.NewDummyVoteproof(
		height,
		base.Round(0),
		base.StageINIT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, avp, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong stage of accept voteproof")
}

func (t *testINIT) TestDRAWACCEPTBaseVoteproofWrongACCEPTVoteproofHeight() {
	height := base.Height(3)
	round := base.Round(33)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round-1,
		base.StageACCEPT,
		base.VoteResultDraw,
	)

	avp := base.NewDummyVoteproof(
		height-2,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, avp, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong height of accept voteproof")
}

func (t *testINIT) TestDRAWACCEPTBaseVoteproofWrongACCEPTVoteproofResult() {
	height := base.Height(3)
	round := base.Round(33)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round-1,
		base.StageACCEPT,
		base.VoteResultDraw,
	)

	avp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultDraw,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, avp, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong result of accept voteproof")
}

func (t *testINIT) TestINITBaseVoteproofEmptyACCEPTVoteproof() {
	height := base.Height(3)
	round := base.Round(33)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round-1,
		base.StageINIT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, nil, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "empty accept voteproof with init base voteproof")
}

func (t *testINIT) TestINITBaseVoteproofWrongHeight() {
	height := base.Height(3)
	round := base.Round(33)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height+1,
		round-1,
		base.StageINIT,
		base.VoteResultMajority,
	)

	avp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, avp, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong height of init ballot + init base voteproof")
}

func (t *testINIT) TestINITBaseVoteproofWrongRound() {
	height := base.Height(3)
	round := base.Round(33)

	fact := t.factSign(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round-2,
		base.StageINIT,
		base.VoteResultMajority,
	)

	avp := base.NewDummyVoteproof(
		height-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, avp, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong round of init ballot + init base voteproof")
}

func (t *testINIT) TestHintNotMatched() {
	fact := t.factSign(base.Height(3), base.Round(0))

	bvp := base.NewDummyVoteproof(
		fact.Height()-1,
		fact.Round(),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bvp, nil, t.pk, t.networkID)
	t.NoError(err)

	sl.BaseSeal.BaseSeal = seal.NewBaseSealWithHint(ProposalHint)
	sl.BaseSeal.BaseSeal.GenerateBodyHashFunc = func() (valuehash.Hash, error) {
		return valuehash.NewSHA256(sl.BodyBytes()), nil
	}

	t.NoError(sl.Sign(t.pk, t.networkID))

	err = sl.IsValid(t.networkID)
	t.NotNil(err)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "ballot has weird fact")
}

func TestINIT(t *testing.T) {
	suite.Run(t, new(testINIT))
}
