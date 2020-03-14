package state

import (
	"testing"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
	"github.com/stretchr/testify/suite"
)

type testStateHintedValueJSON struct {
	suite.Suite

	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testStateHintedValueJSON) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewJSONEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(dummy{})
	_ = t.encs.AddHinter(HintedValue{})
}

func (t *testStateHintedValueJSON) TestEncode() {
	d := dummy{}
	d.v = 33

	bv, err := NewHintedValue(d)
	t.NoError(err)

	b, err := util.JSONMarshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(HintedValue).v)
}

func (t *testStateHintedValueJSON) TestEmpty() {
	var d dummy
	bv, err := NewHintedValue(d)
	t.NoError(err)

	b, err := util.JSONMarshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(HintedValue).v)
}

func TestStateHintedValueJSON(t *testing.T) {
	suite.Run(t, new(testStateHintedValueJSON))
}
