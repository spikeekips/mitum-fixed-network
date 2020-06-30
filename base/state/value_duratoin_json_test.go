package state

import (
	"testing"
	"time"

	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/valuehash"
	"github.com/stretchr/testify/suite"
)

type testStateDurationValueJSON struct {
	suite.Suite

	encs *encoder.Encoders
	enc  encoder.Encoder
}

func (t *testStateDurationValueJSON) SetupSuite() {
	t.encs = encoder.NewEncoders()
	t.enc = jsonenc.NewEncoder()
	_ = t.encs.AddEncoder(t.enc)

	_ = t.encs.AddHinter(valuehash.SHA256{})
	_ = t.encs.AddHinter(dummy{})
	_ = t.encs.AddHinter(DurationValue{})
}

func (t *testStateDurationValueJSON) TestCases() {
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

				b, err := jsonenc.Marshal(iv)
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

func TestStateDurationValueJSON(t *testing.T) {
	suite.Run(t, new(testStateDurationValueJSON))
}
