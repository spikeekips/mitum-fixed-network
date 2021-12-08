package ballot

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testACCEPTFactEncode struct {
	baseTest
	enc encoder.Encoder
}

func (t *testACCEPTFactEncode) SetupSuite() {
	t.baseTest.SetupSuite()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.TestAddHinter(base.StringAddressHinter))
	t.NoError(encs.TestAddHinter(base.SignedBallotFactHinter))
	t.NoError(encs.TestAddHinter(ACCEPTFactHinter))
}

func (t *testACCEPTFactEncode) TestEncode() {
	fact := NewACCEPTFact(
		base.Height(3),
		base.Round(0),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	)
	fact.BaseFact = NewBaseFact(hint.NewHint(base.ACCEPTBallotFactType, "v0.0.3"), fact.Height(), fact.Round())
	fact.BaseFact.h = valuehash.NewSHA256(fact.bytes())

	b, err := t.enc.Marshal(fact)
	t.NoError(err)

	ht, err := t.enc.Decode(b)
	t.NoError(err)

	ufact, ok := ht.(base.BallotFact)
	t.True(ok)

	t.NoError(ufact.IsValid(nil))

	t.compareFact(fact, ufact)
}

func TestACCEPTFactEncodeJSON(t *testing.T) {
	b := new(testACCEPTFactEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestACCEPTFactEncodeBSON(t *testing.T) {
	b := new(testACCEPTFactEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}

type testACCEPTEncode struct {
	baseTest
	enc encoder.Encoder
}

func (t *testACCEPTEncode) SetupSuite() {
	t.baseTest.SetupSuite()

	encs := encoder.NewEncoders()
	t.NoError(encs.AddEncoder(t.enc))
	t.NoError(encs.TestAddHinter(base.StringAddressHinter))
	t.NoError(encs.TestAddHinter(key.BasePublickey{}))
	t.NoError(encs.TestAddHinter(base.BaseFactSignHinter))
	_ = encs.TestAddHinter(base.BallotFactSignHinter)
	t.NoError(encs.TestAddHinter(base.DummyVoteproof{}))
	t.NoError(encs.TestAddHinter(base.SignedBallotFactHinter))
	t.NoError(encs.TestAddHinter(ACCEPTFactHinter))
	t.NoError(encs.TestAddHinter(ACCEPTHinter))
}

func (t *testACCEPTEncode) TestEncode() {
	fact := NewACCEPTFact(
		base.Height(3),
		base.Round(0),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(),
	)
	t.NoError(fact.IsValid(nil))

	bavp := base.NewDummyVoteproof(
		fact.Height(),
		fact.Round(),
		base.StageINIT,
		base.VoteResultMajority,
	)

	sl, err := NewACCEPT(fact, base.RandomStringAddress(), bavp, t.pk, t.networkID)
	t.NoError(err)
	t.NoError(sl.IsValid(t.networkID))

	b, err := t.enc.Marshal(sl)
	t.NoError(err)

	usl, err := base.DecodeBallot(b, t.enc)
	t.NoError(err)
	t.NotNil(usl)

	t.compareBallot(sl, usl)

	ub, err := t.enc.Marshal(usl)
	t.NoError(err)
	t.Equal(b, ub)
}

func TestACCEPTEncodeJSON(t *testing.T) {
	b := new(testACCEPTEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestACCEPTEncodeBSON(t *testing.T) {
	b := new(testACCEPTFactEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
