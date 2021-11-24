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

type testACCEPTFact struct {
	suite.Suite
}

func (t *testACCEPTFact) TestNew() {
	height := base.Height(3)
	round := base.Round(33)
	proposal := valuehash.RandomSHA256()
	newBlock := valuehash.RandomSHA256()

	fact := NewACCEPTFact(
		height,
		round,
		proposal,
		newBlock,
	)
	t.NoError(fact.IsValid(nil))

	_, ok := (interface{})(fact).(base.ACCEPTBallotFact)
	t.True(ok)

	t.Equal(height, fact.Height())
	t.Equal(round, fact.Round())
	t.True(proposal.Equal(fact.Proposal()))
	t.True(newBlock.Equal(fact.NewBlock()))
}

func (t *testACCEPTFact) TestEmptyProposal() {
	fact := NewACCEPTFact(
		base.Height(3),
		base.Round(33),
		nil,
		valuehash.RandomSHA256(),
	)

	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testACCEPTFact) TestEmptyNewBlock() {
	fact := NewACCEPTFact(
		base.Height(3),
		base.Round(33),
		valuehash.RandomSHA256(),
		nil,
	)

	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testACCEPTFact) TestHashNotMatched() {
	fact := NewACCEPTFact(
		base.Height(3),
		base.Round(33),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	)
	t.NoError(fact.IsValid(nil))

	fact.h = valuehash.RandomSHA256()
	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "hash does not match")
}

func TestACCEPTFact(t *testing.T) {
	suite.Run(t, new(testACCEPTFact))
}

type testACCEPT struct {
	testSuite
}

func (t *testACCEPT) SetupSuite() {
	t.testSuite.SetupSuite()

	t.newFact = func(height base.Height, round base.Round) base.BallotFact {
		return NewACCEPTFact(
			height,
			round,
			valuehash.RandomSHA256(),
			valuehash.RandomSHA256(),
		)
	}

	t.newBallot = func(
		fact base.BallotFact,
		n base.Address,
		bvp base.Voteproof,
		avp base.Voteproof,
	) (base.Ballot, error) {
		sl, err := NewACCEPT(fact.(ACCEPTFact), n, bvp, t.pk, t.networkID)
		if err != nil {
			return nil, err
		}

		sl.BaseSeal.acceptVoteproof = avp
		if err := sl.Sign(t.pk, t.networkID); err != nil {
			return nil, err
		}

		return sl, nil
	}
}

func (t *testACCEPT) TestSign() {
	height := base.Height(3)
	round := base.Round(0)

	fact := t.newFact(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round,
		base.StageINIT,
		base.VoteResultMajority,
	)

	sl, err := NewACCEPT(fact.(ACCEPTFact), base.RandomStringAddress(), bvp, t.pk, t.networkID)
	t.NoError(err)

	t.Implements((*seal.Seal)(nil), sl)

	_ = (interface{})(sl).(base.Ballot)
	t.Implements((*base.Ballot)(nil), sl)

	_ = (interface{})(sl).(base.ACCEPTBallot)
	t.Implements((*base.ACCEPTBallot)(nil), sl)

	t.NoError(sl.IsValid(t.networkID))

	t.compareFact(fact, sl.Fact())
	t.Equal(sl.Fact().Stage(), base.StageACCEPT)
	t.True(sl.Hint().Equal(ACCEPTHint))

	t.testSignWithFact(&sl, sl.FactSign().Node())
}

func TestACCEPT(t *testing.T) {
	suite.Run(t, new(testACCEPT))
}
