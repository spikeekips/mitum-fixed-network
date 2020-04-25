package state

import (
	"testing"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

type testStateNumberValueBSON struct {
	suite.Suite

	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testStateNumberValueBSON) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = encoder.NewBSONEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(dummy{})
	_ = t.encs.AddHinter(NumberValue{})
}

func (t *testStateNumberValueBSON) TestEncode() {
	iv, err := NewNumberValue(int64(33))
	t.NoError(err)

	b, err := bson.Marshal(iv)
	t.NoError(err)

	decoded, err := t.enc.DecodeByHint(b)
	t.NoError(err)
	t.Implements((*Value)(nil), decoded)

	u := decoded.(Value)

	t.True(iv.Hint().Equal(u.Hint()))
	t.True(iv.Equal(u))
	t.Equal(iv.v, u.(NumberValue).v)
}

func (t *testStateNumberValueBSON) TestCases() {
	cases := []struct {
		name string
		v    interface{}
		err  string
	}{
		{name: "int", v: 34},
		{name: "int8", v: int8(34)},
		{name: "int16", v: int16(34)},
		{name: "int32", v: int32(34)},
		{name: "int64", v: int64(34)},
		{name: "uint", v: 34},
		{name: "uint8", v: uint8(34)},
		{name: "uint16", v: uint16(34)},
		{name: "uint32", v: uint32(34)},
		{name: "uint64", v: uint64(34)},
		{name: "float64", v: float64(34)},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				iv, err := NewNumberValue(c.v)
				t.NoError(err, "%d: name=%s value=%s", i, c.name, c.v)

				b, err := bson.Marshal(iv)
				t.NoError(err, "%d: name=%s value=%s", i, c.name, c.v)

				decoded, err := t.enc.DecodeByHint(b)
				t.NoError(err, "%d: name=%s value=%s", i, c.name, c.v)
				t.Implements((*Value)(nil), decoded)

				u, ok := decoded.(NumberValue)
				t.True(ok, "%d: name=%s value=%s", i, c.name, c.v)

				t.Equal(c.v, u.v, "%d: name=%s value=%s", i, c.name, c.v)
				t.True(iv.Hash().Equal(u.Hash()), "%d: name=%s value=%s", i, c.name, c.v)
				t.Equal(iv.b, u.b, "%d: name=%s value=%s", i, c.name, c.v)
			},
		)
	}
}

func TestStateNumberValueBSON(t *testing.T) {
	suite.Run(t, new(testStateNumberValueBSON))
}
