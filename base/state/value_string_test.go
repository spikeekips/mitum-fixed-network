package state

import (
	"testing"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testStateStringValue struct {
	suite.Suite
}

func (t *testStateStringValue) TestCases() {
	cases := []struct {
		name     string
		v        interface{}
		expected string
		err      string
	}{
		{name: "string", v: "show me"},
		{name: "empty string", v: ""},
		{name: "int", v: 10, err: "not string-like"},
		{
			name:     "String()",
			v:        valuehash.NewSHA256([]byte("eat me")),
			expected: valuehash.NewSHA256([]byte("eat me")).String(),
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				sv, err := NewStringValue(c.v)
				if len(c.err) > 0 {
					if err == nil {
						t.NoError(xerrors.Errorf("error expected, but got %v", c.err), "%d: %v; %v != %v", i, c.name, c.err, err)
						return
					}

					t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
					return
				}

				if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
					return
				}

				t.NotNil(sv)

				var expected interface{}
				if len(c.expected) > 0 {
					expected = c.expected
				} else {
					expected = c.v
				}

				t.Equal(expected, sv.Interface())
			},
		)
	}
}

func TestStateStringValue(t *testing.T) {
	suite.Run(t, new(testStateStringValue))
}
