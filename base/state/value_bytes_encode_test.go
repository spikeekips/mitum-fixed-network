package state

import (
	"testing"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testStateBytesValueEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testStateBytesValueEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(t.enc)

	_ = encs.TestAddHinter(BytesValueHinter)
}

func (t *testStateBytesValueEncode) TestEncode() {
	v := []byte("showme")
	bv, err := NewBytesValue(v)
	t.NoError(err)

	b, err := t.enc.Marshal(bv)
	t.NoError(err)

	decoded, err := t.enc.Decode(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(BytesValue).v)
}

func (t *testStateBytesValueEncode) TestEmpty() {
	var v []byte

	bv, err := NewBytesValue(v)
	t.NoError(err)

	b, err := t.enc.Marshal(bv)
	t.NoError(err)

	decoded, err := t.enc.Decode(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(BytesValue).v)
}

func TestStateBytesValueEncodeJSON(t *testing.T) {
	b := new(testStateBytesValueEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestStateBytesValueEncodeBSON(t *testing.T) {
	b := new(testStateBytesValueEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
