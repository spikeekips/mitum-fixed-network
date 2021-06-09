package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type testStateDurationValueEncode struct {
	suite.Suite

	enc encoder.Encoder
}

func (t *testStateDurationValueEncode) SetupSuite() {
	encs := encoder.NewEncoders()
	_ = encs.AddEncoder(t.enc)

	_ = encs.TestAddHinter(dummy{})
	_ = encs.TestAddHinter(DurationValue{})
}

func (t *testStateDurationValueEncode) TestCases() {
	cases := []struct {
		name string
		v    time.Duration
		err  string
	}{
		{name: "seconds", v: time.Second * 133},
		{name: "milliseconds", v: time.Millisecond * 133},
		{name: "nanoseconds", v: time.Nanosecond * 133},
		{name: "negative seconds", v: time.Second * -133},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				iv, err := NewDurationValue(c.v)
				t.NoError(err, "%d: name=%s value=%s", i, c.name, c.v)

				b, err := t.enc.Marshal(iv)
				t.NoError(err, "%d: name=%s value=%s", i, c.name, c.v)

				decoded, err := t.enc.DecodeByHint(b)
				t.NoError(err, "%d: name=%s value=%s", i, c.name, c.v)
				t.Implements((*Value)(nil), decoded)

				u, ok := decoded.(DurationValue)
				t.True(ok, "%d: name=%s value=%s", i, c.name, c.v)

				t.Equal(c.v, u.v, "%d: name=%s value=%s", i, c.name, c.v)
				t.True(iv.Hash().Equal(u.Hash()), "%d: name=%s value=%s", i, c.name, c.v)
			},
		)
	}
}

func TestStateDurationValueEncodeJSON(t *testing.T) {
	b := new(testStateDurationValueEncode)
	b.enc = jsonenc.NewEncoder()

	suite.Run(t, b)
}

func TestStateDurationValueEncodeBSON(t *testing.T) {
	b := new(testStateDurationValueEncode)
	b.enc = bsonenc.NewEncoder()

	suite.Run(t, b)
}
