package hint

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
)

type testType struct {
	suite.Suite
}

func (t *testType) TestNew() {
	cases := []struct {
		name string
		s    string
		err  string
	}{
		{name: "valid", s: "showme"},
		{name: "blank head", s: " showme", err: "invalid char found"},
		{name: "blank tail", s: "showme ", err: "invalid char found"},
		{name: "uppercase", s: "shOwme", err: "invalid char found"},
		{name: "slash", s: "sh/wme", err: "invalid char found"},
		{name: "hyphen", s: "sh-wme"},
		{name: "underscore", s: "sh-w_me"},
		{name: "plus", s: "sh-w_m+e"},
		{name: "empty", s: "", err: "empty Type"},
		{name: "too long", s: strings.Repeat("a", MaxTypeLength+1), err: "Type too long"},
		{name: "2 chars", s: "sa"},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				ty := Type(c.s)
				err := ty.IsValid(nil)
				if len(c.err) > 0 {
					if err == nil {
						t.NoError(errors.Errorf("expected %q, but nil error", c.err), "%d: %v", i, c.name)

						return
					}

					t.Contains(err.Error(), c.err, "%d: %v", i, c.name)
				} else if err != nil {
					t.NoError(errors.Errorf("expected nil error, but %+v", err), "%d: %v", i, c.name)

					return
				}
			},
		)
	}
}

func TestType(t *testing.T) {
	suite.Run(t, new(testType))
}
