package base

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

type testStringAddress struct {
	suite.Suite
}

func (t *testStringAddress) TestEmpty() {
	_, err := NewStringAddress("")
	t.Error(err)
}

func (t *testStringAddress) TestFormat() {
	uuidString := util.UUID().String()

	cases := []struct {
		name     string
		s        string
		expected string
		err      string
	}{
		{
			name:     "uuid",
			s:        uuidString,
			expected: hint.NewHintedString(StringAddressHint, uuidString).String(),
		},
		{
			name: "blank first",
			s:    " showme",
			err:  "has blank",
		},
		{
			name: "blank inside",
			s:    "sh owme",
			err:  "has blank",
		},
		{
			name: "blank ends",
			s:    "showme ",
			err:  "has blank",
		},
		{
			name: "blank ends, tab",
			s:    "showme\t",
			err:  "has blank",
		},
		{
			name:     "has underscore",
			s:        "showm_e",
			expected: hint.NewHintedString(StringAddressHint, "showm_e").String(),
		},
		{
			name: "has plus sign",
			s:    "showm+e",
			err:  "invalid address string",
		},
		{
			name: "has at sign",
			s:    "showm@e",
			err:  "invalid address string",
		},
		{
			name: "has dot",
			s:    "showm.e",
			err:  "invalid address string",
		},
		{
			name: "has dot #1",
			s:    "showme.",
			err:  "invalid address string",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				r, err := NewStringAddress(c.s)
				if err != nil {
					if len(c.err) < 1 {
						t.NoError(err)
					} else {
						t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
					}
				} else if len(c.err) > 0 {
					t.NoError(errors.Errorf(c.err))
				} else {
					t.Equal(c.expected, r.String(), "%d: %v; %v != %v", i, c.name, c.expected, r)
					t.Equal(c.s, r.Raw(), "%d: %v; %v != %v", i, c.name, c.s, r.Raw())
				}
			},
		)
	}
}

func (t *testStringAddress) TestHasBlank() {
	_, err := NewStringAddress("a b")
	t.Contains(err.Error(), "has blank")

	_, err = NewStringAddress("ab\t-")
	t.Contains(err.Error(), "has blank")
}

func (t *testStringAddress) TestString() {
	sa, err := NewStringAddress(util.UUID().String())
	t.NoError(err)

	una, err := NewStringAddressFromHintedString(sa.String())
	t.NoError(err)

	t.True(sa.Equal(una))
}

func (t *testStringAddress) TestJSON() {
	sa, err := NewStringAddress(util.UUID().String())
	t.NoError(err)

	b, err := util.JSON.Marshal(sa)
	t.NoError(err)

	var s string
	t.NoError(util.JSON.Unmarshal(b, &s))

	usa, err := NewStringAddressFromHintedString(s)
	t.NoError(err)

	t.True(sa.Equal(usa))
}

func (t *testStringAddress) TestBSON() {
	sa, err := NewStringAddress(util.UUID().String())
	t.NoError(err)

	b, err := bsonenc.Marshal(struct {
		N StringAddress
	}{N: sa})
	t.NoError(err)

	var una struct {
		N bson.RawValue
	}

	t.NoError(bsonenc.Unmarshal(b, &una))

	usa, err := NewStringAddressFromHintedString(string(una.N.StringValue()))
	t.NoError(err)

	t.True(sa.Equal(usa))
}

func TestStringAddress(t *testing.T) {
	suite.Run(t, new(testStringAddress))
}
