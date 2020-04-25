package state

import (
	"testing"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

type testStateHintedValueBSON struct {
	suite.Suite

	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testStateHintedValueBSON) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewBSONEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(dummy{})
	_ = t.encs.AddHinter(HintedValue{})
}

func (t *testStateHintedValueBSON) TestEncode() {
	d := dummy{}
	d.v = 33

	bv, err := NewHintedValue(d)
	t.NoError(err)

	b, err := bson.Marshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(HintedValue).v)
}

func (t *testStateHintedValueBSON) TestEmpty() {
	var d dummy
	bv, err := NewHintedValue(d)
	t.NoError(err)

	b, err := bson.Marshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(HintedValue).v)
}

func TestStateHintedValueBSON(t *testing.T) {
	suite.Run(t, new(testStateHintedValueBSON))
}
