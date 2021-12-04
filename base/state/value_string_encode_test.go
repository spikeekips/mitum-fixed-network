package state

import (
	"testing"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testStateStringValueEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testStateStringValueEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(t.enc)

	_ = encs.TestAddHinter(StringValueHinter)
}

func (t *testStateStringValueEncode) TestEncode() {
	sv, err := NewStringValue("showme")
	t.NoError(err)

	b, err := t.enc.Marshal(sv)
	t.NoError(err)

	decoded, err := t.enc.Decode(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(sv.Hint().Equal(u.Hint()))
	t.True(sv.Equal(u))
	t.Equal(sv.v, u.(StringValue).v)
}

func TestStateStringValueEncodeJSON(t *testing.T) {
	b := new(testStateStringValueEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestStateStringValueEncodeBSON(t *testing.T) {
	b := new(testStateStringValueEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
