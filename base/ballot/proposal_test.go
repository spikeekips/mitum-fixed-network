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

type testProposalFact struct {
	suite.Suite
	ops []valuehash.Hash
}

func (t *testProposalFact) SetupTest() {
	t.ops = []valuehash.Hash{valuehash.RandomSHA256(), valuehash.RandomSHA256()}
}

func (t *testProposalFact) TestNew() {
	height := base.Height(3)
	round := base.Round(33)
	n := base.RandomStringAddress()

	fact := NewProposalFact(
		height,
		round,
		n,
		t.ops,
	)
	t.NoError(fact.IsValid(nil))

	_, ok := (interface{})(fact).(base.ProposalFact)
	t.True(ok)

	t.Equal(height, fact.Height())
	t.Equal(round, fact.Round())
	t.Equal(len(t.ops), len(fact.Operations()))

	for i := range t.ops {
		t.True(t.ops[i].Equal(fact.Operations()[i]))
	}
}

func (t *testProposalFact) TestEmptyHashInOperations() {
	fact := NewProposalFact(
		base.Height(3),
		base.Round(33),
		base.RandomStringAddress(),
		[]valuehash.Hash{valuehash.RandomSHA256(), valuehash.RandomSHA256(), nil},
	)

	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testProposalFact) TestEmptyOperations() {
	fact := NewProposalFact(
		base.Height(3),
		base.Round(33),
		base.RandomStringAddress(),
		nil,
	)

	t.NoError(fact.IsValid(nil))
}

func (t *testProposalFact) TestHashNotMatched() {
	fact := NewProposalFact(
		base.Height(3),
		base.Round(33),
		base.RandomStringAddress(),
		t.ops,
	)
	t.NoError(fact.IsValid(nil))

	fact.h = valuehash.RandomSHA256()
	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "hash does not match")
}

func (t *testProposalFact) TestEmptyProposer() {
	height := base.Height(3)
	round := base.Round(0)

	fact := NewProposalFact(
		height,
		round,
		nil,
		[]valuehash.Hash{valuehash.RandomSHA256(), valuehash.RandomSHA256()},
	)

	err := fact.IsValid(nil)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func TestProposalFact(t *testing.T) {
	suite.Run(t, new(testProposalFact))
}

type testProposal struct {
	testSuite
}

func (t *testProposal) SetupSuite() {
	t.testSuite.SetupSuite()

	t.newFact = func(height base.Height, round base.Round) base.BallotFact {
		return NewProposalFact(
			height,
			round,
			base.RandomStringAddress(),
			[]valuehash.Hash{valuehash.RandomSHA256(), valuehash.RandomSHA256()},
		)
	}

	t.newBallot = func(
		fact base.BallotFact,
		_ base.Address,
		bvp base.Voteproof,
		avp base.Voteproof,
	) (base.Ballot, error) {
		sl, err := NewProposal(fact.(ProposalFact), fact.(ProposalFact).proposer, bvp, t.pk, t.networkID)
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

func (t *testProposal) TestSign() {
	height := base.Height(3)
	round := base.Round(0)

	fact := t.newFact(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round,
		base.StageINIT,
		base.VoteResultMajority,
	)

	sl, err := NewProposal(fact.(ProposalFact), fact.(ProposalFact).proposer, bvp, t.pk, t.networkID)
	t.NoError(err)

	t.Implements((*seal.Seal)(nil), sl)

	_ = (interface{})(sl).(base.Ballot)
	t.Implements((*base.Ballot)(nil), sl)

	_ = (interface{})(sl).(base.Proposal)
	t.Implements((*base.Proposal)(nil), sl)

	t.NoError(sl.IsValid(t.networkID))

	t.compareFact(fact, sl.Fact())
	t.Equal(sl.Fact().Stage(), base.StageProposal)
	t.True(sl.Hint().Equal(ProposalHint))

	t.testSignWithFact(&sl, sl.Fact().Proposer())
}

func (t *testProposal) TestDifferentProposer() {
	height := base.Height(3)
	round := base.Round(0)

	fact := t.newFact(height, round)

	bvp := base.NewDummyVoteproof(
		height,
		round,
		base.StageINIT,
		base.VoteResultMajority,
	)

	sl, err := NewProposal(fact.(ProposalFact), base.RandomStringAddress(), bvp, t.pk, t.networkID)
	t.NoError(err)

	err = sl.IsValid(t.networkID)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "proposal fact is not signed by factsign node")
}

func TestProposal(t *testing.T) {
	suite.Run(t, new(testProposal))
}
