package state

import (
	"testing"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
)

type testStateSliceValueJSON struct {
	suite.Suite

	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testStateSliceValueJSON) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewJSONEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(dummy{})
	_ = t.encs.AddHinter(SliceValue{})
}

func (t *testStateSliceValueJSON) TestEncode() {
	d := dummy{}
	d.v = 33

	bv, err := NewSliceValue([]hint.Hinter{d})
	t.NoError(err)

	b, err := util.JSONMarshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(SliceValue).v)
}

func TestStateSliceValueJSON(t *testing.T) {
	suite.Run(t, new(testStateSliceValueJSON))
}
