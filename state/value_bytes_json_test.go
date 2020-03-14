package state

import (
	"testing"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"github.com/stretchr/testify/suite"
)

type testStateBytesValueJSON struct {
	suite.Suite

	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testStateBytesValueJSON) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewJSONEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(BytesValue{})
}

func (t *testStateBytesValueJSON) TestEncode() {
	v := []byte("showme")
	bv, err := NewBytesValue(v)
	t.NoError(err)

	b, err := util.JSONMarshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(BytesValue).v)
}

func (t *testStateBytesValueJSON) TestEmpty() {
	var v []byte

	bv, err := NewBytesValue(v)
	t.NoError(err)

	b, err := util.JSONMarshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(BytesValue).v)
}

func TestStateBytesValueJSON(t *testing.T) {
	suite.Run(t, new(testStateBytesValueJSON))
}
