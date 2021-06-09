package state

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type testStateStringValueEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testStateStringValueEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(t.enc)

	_ = encs.TestAddHinter(StringValue{})
}

func (t *testStateStringValueEncode) TestEncode() {
	sv, err := NewStringValue("showme")
	t.NoError(err)

	b, err := t.enc.Marshal(sv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
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
