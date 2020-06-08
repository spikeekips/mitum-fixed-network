package state

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

type testStateStringValueBSON struct {
	suite.Suite

	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testStateStringValueBSON) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = bsonenc.NewEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(StringValue{})
}

func (t *testStateStringValueBSON) TestEncode() {
	sv, err := NewStringValue("showme")
	t.NoError(err)

	b, err := bsonenc.Marshal(sv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(sv.Hint().Equal(u.Hint()))
	t.True(sv.Equal(u))
	t.Equal(sv.v, u.(StringValue).v)
}

func TestStateStringValueBSON(t *testing.T) {
	suite.Run(t, new(testStateStringValueBSON))
}
