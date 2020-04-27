package state

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/valuehash"
)

type testStateBytesValue struct {
	suite.Suite
}

func (t *testStateBytesValue) TestCases() {
	cases := []struct {
		name     string
		v        interface{}
		expected []byte
		err      string
	}{
		{name: "string", v: "show me", expected: []byte("show me")},
		{name: "empty string", v: "", expected: []byte("")},
		{name: "int", v: 10, err: "not bytes-like"},
		{
			name:     "Bytes()",
			v:        valuehash.NewSHA256([]byte("eat me")),
			expected: valuehash.NewSHA256([]byte("eat me")).Bytes(),
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				sv, err := NewBytesValue(c.v)
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
				if c.expected != nil {
					expected = c.expected
				} else {
					expected = c.v
				}

				t.Equal(expected, sv.Interface(), "%d: %v; %v != %v", i, c.name, expected, sv.Interface())
			},
		)
	}
}

func TestStateBytesValue(t *testing.T) {
	suite.Run(t, new(testStateBytesValue))
}
