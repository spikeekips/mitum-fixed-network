package ballot

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testProposalFactEncode struct {
	baseTest
	enc encoder.Encoder
}

func (t *testProposalFactEncode) SetupSuite() {
	t.baseTest.SetupSuite()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.TestAddHinter(base.StringAddressHinter))
	t.NoError(encs.TestAddHinter(base.SignedBallotFactHinter))
	t.NoError(encs.TestAddHinter(ProposalFactHinter))
}

func (t *testProposalFactEncode) TestEncode() {
	fact := NewProposalFact(
		base.Height(3),
		base.Round(0),
		base.RandomStringAddress(),
		[]valuehash.Hash{valuehash.RandomSHA256(), valuehash.RandomSHA256()},
	)

	b, err := t.enc.Marshal(fact)
	t.NoError(err)

	ht, err := t.enc.Decode(b)
	t.NoError(err)

	ufact, ok := ht.(base.BallotFact)
	t.True(ok)

	t.NoError(ufact.IsValid(nil))

	t.compareFact(fact, ufact)
}

func TestProposalFactEncodeJSON(t *testing.T) {
	b := new(testProposalFactEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestProposalFactEncodeBSON(t *testing.T) {
	b := new(testProposalFactEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}

type testProposalEncode struct {
	baseTest
	enc encoder.Encoder
}

func (t *testProposalEncode) SetupSuite() {
	t.baseTest.SetupSuite()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.TestAddHinter(base.StringAddressHinter))
	t.NoError(encs.TestAddHinter(key.BasePublickey{}))
	t.NoError(encs.TestAddHinter(base.BaseFactSignHinter))
	_ = encs.TestAddHinter(base.BallotFactSignHinter)
	t.NoError(encs.TestAddHinter(base.DummyVoteproof{}))
	t.NoError(encs.TestAddHinter(base.SignedBallotFactHinter))
	t.NoError(encs.TestAddHinter(ProposalFactHinter))
	t.NoError(encs.TestAddHinter(ProposalHinter))
}

func (t *testProposalEncode) TestEncode() {
	fact := NewProposalFact(
		base.Height(3),
		base.Round(0),
		base.RandomStringAddress(),
		[]valuehash.Hash{valuehash.RandomSHA256(), valuehash.RandomSHA256()},
	)
	t.NoError(fact.IsValid(nil))

	bavp := base.NewDummyVoteproof(
		fact.Height(),
		fact.Round(),
		base.StageINIT,
		base.VoteResultMajority,
	)

	sl, err := NewProposal(fact, fact.proposer, bavp, t.pk, t.networkID)
	t.NoError(err)
	t.NoError(sl.IsValid(t.networkID))

	b, err := t.enc.Marshal(sl)
	t.NoError(err)

	var usl base.Ballot
	t.NoError(encoder.Decode(b, t.enc, &usl))
	t.NotNil(usl)

	t.compareBallot(sl, usl)

	ub, err := t.enc.Marshal(usl)
	t.NoError(err)
	t.Equal(b, ub)
}

func TestProposalEncodeJSON(t *testing.T) {
	b := new(testProposalEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestProposalEncodeBSON(t *testing.T) {
	b := new(testProposalFactEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
