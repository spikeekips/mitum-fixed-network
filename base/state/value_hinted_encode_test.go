package state

import (
	"testing"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/stretchr/testify/suite"
)

type testStateHintedValueEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testStateHintedValueEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(t.enc)

	_ = encs.TestAddHinter(dummy{})
	_ = encs.TestAddHinter(HintedValueHinter)
}

func (t *testStateHintedValueEncode) TestEncode() {
	d := dummy{}
	d.v = 33

	bv, err := NewHintedValue(d)
	t.NoError(err)

	b, err := t.enc.Marshal(bv)
	t.NoError(err)

	decoded, err := t.enc.Decode(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(HintedValue).v)
}

func (t *testStateHintedValueEncode) TestEmpty() {
	var d dummy
	bv, err := NewHintedValue(d)
	t.NoError(err)

	b, err := t.enc.Marshal(bv)
	t.NoError(err)

	decoded, err := t.enc.Decode(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(HintedValue).v)
}

func TestStateHintedValueEncodeJSON(t *testing.T) {
	b := new(testStateHintedValueEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestStateHintedValueEncodeBSON(t *testing.T) {
	b := new(testStateHintedValueEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
