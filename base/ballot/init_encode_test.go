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

type testINITFactEncode struct {
	baseTest
	enc encoder.Encoder
}

func (t *testINITFactEncode) SetupSuite() {
	t.baseTest.SetupSuite()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.TestAddHinter(base.StringAddressHinter))
	t.NoError(encs.TestAddHinter(base.SignedBallotFactHinter))
	t.NoError(encs.TestAddHinter(INITFactHinter))
}

func (t *testINITFactEncode) TestEncode() {
	fact := NewINITFact(
		base.Height(3),
		base.Round(0),
		valuehash.RandomSHA256(),
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

func TestINITFactEncodeJSON(t *testing.T) {
	b := new(testINITFactEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestINITFactEncodeBSON(t *testing.T) {
	b := new(testINITFactEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}

type testINITEncode struct {
	baseTest
	enc encoder.Encoder
}

func (t *testINITEncode) SetupSuite() {
	t.baseTest.SetupSuite()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.TestAddHinter(base.StringAddressHinter))
	t.NoError(encs.TestAddHinter(key.BasePublickey{}))
	t.NoError(encs.TestAddHinter(base.BaseFactSignHinter))
	_ = encs.TestAddHinter(base.BallotFactSignHinter)
	t.NoError(encs.TestAddHinter(base.DummyVoteproof{}))
	t.NoError(encs.TestAddHinter(base.SignedBallotFactHinter))
	t.NoError(encs.TestAddHinter(INITFactHinter))
	t.NoError(encs.TestAddHinter(INITHinter))
}

func (t *testINITEncode) TestEncode() {
	fact := NewINITFact(
		base.Height(3),
		base.Round(0),
		valuehash.RandomSHA256(),
	)
	t.NoError(fact.IsValid(nil))

	bavp := base.NewDummyVoteproof(
		fact.Height()-1,
		base.Round(0),
		base.StageACCEPT,
		base.VoteResultMajority,
	)

	sl, err := NewINIT(fact, base.RandomStringAddress(), bavp, nil, t.pk, t.networkID)
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

func TestINITEncodeJSON(t *testing.T) {
	b := new(testINITEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestINITEncodeBSON(t *testing.T) {
	b := new(testINITFactEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
