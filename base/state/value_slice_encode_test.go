package state

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

type testStateSliceValueEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testStateSliceValueEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(t.enc)

	_ = encs.TestAddHinter(dummy{})
	_ = encs.TestAddHinter(SliceValue{})
}

func (t *testStateSliceValueEncode) TestEncode() {
	d := dummy{}
	d.v = 33

	bv, err := NewSliceValue([]hint.Hinter{d})
	t.NoError(err)

	b, err := t.enc.Marshal(bv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(bv.Hint().Equal(u.Hint()))
	t.True(bv.Equal(u))
	t.Equal(bv.v, u.(SliceValue).v)
}

func TestStateSliceValueEncodeJSON(t *testing.T) {
	b := new(testStateSliceValueEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestStateSliceValueEncodeBSON(t *testing.T) {
	b := new(testStateSliceValueEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
