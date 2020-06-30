package state

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testStateSliceValueBSON struct {
	suite.Suite

	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testStateSliceValueBSON) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = bsonenc.NewEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(dummy{})
	_ = t.encs.AddHinter(SliceValue{})
}

func (t *testStateSliceValueBSON) TestEncode() {
	d := dummy{}
	d.v = 33

	bv, err := NewSliceValue([]hint.Hinter{d})
	t.NoError(err)

	b, err := bsonenc.Marshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(SliceValue).v)
}

func TestStateSliceValueBSON(t *testing.T) {
	suite.Run(t, new(testStateSliceValueBSON))
}
