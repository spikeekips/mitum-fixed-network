package state

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/stretchr/testify/suite"
)

type testStateBytesValueBSON struct {
	suite.Suite

	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testStateBytesValueBSON) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewBSONEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(BytesValue{})
}

func (t *testStateBytesValueBSON) TestEncode() {
	v := []byte("showme")
	bv, err := NewBytesValue(v)
	t.NoError(err)

	b, err := bson.Marshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(BytesValue).v)
}

func (t *testStateBytesValueBSON) TestEmpty() {
	var v []byte

	bv, err := NewBytesValue(v)
	t.NoError(err)

	b, err := bson.Marshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(BytesValue).v)
}

func TestStateBytesValueBSON(t *testing.T) {
	suite.Run(t, new(testStateBytesValueBSON))
}
