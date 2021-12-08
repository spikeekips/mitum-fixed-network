package hint

import (
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

type testHintedString struct {
	suite.Suite
}

func (t *testHintedString) TestParse() {
	cases := []struct {
		name     string
		s        string
		expected string
		err      string
	}{
		{
			name:     "valid",
			s:        "findme~showme-v1.2.3+incompatible",
			expected: "findme~showme-v1.2.3+incompatible",
		},
		{
			name: "empty raw string",
			s:    "~showme-v1.2.3+incompatible",
			err:  "invalid HintedString format",
		},
		{
			name: "empty raw string #1",
			s:    "   ~showme-v1.2.3+incompatible",
			err:  "invalid HintedString format",
		},
		{
			name: "no delimiter",
			s:    "findmeshowme-v1.2.3+incompatible",
			err:  "invalid HintedString format",
		},
		{
			name: "invalid type",
			s:    "findme~sho;wme-v1.2.3incompatible",
			err:  "invalid Hint",
		},
		{
			name: "invalid version",
			s:    "findme~showme-v1.2.3incompatible",
			err:  "invalid version",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				ht, err := ParseHintedString(c.s)
				if len(c.err) > 0 {
					if err == nil {
						t.NoError(errors.Errorf("expected %q, but nil error", c.err), "%d: %v", i, c.name)

						return
					}

					t.Contains(err.Error(), c.err, "%d: %v", i, c.name)

					return
				} else if err != nil {
					t.NoError(errors.Errorf("expected nil error, but %+v", err), "%d: %v", i, c.name)

					return
				}

				t.Equal(c.expected, ht.String(), "%d: %v", i, c.name)
			},
		)
	}
}

func (t *testHintedString) TestEncodeJSON() {
	ht, err := ParseHintedString("showme~findme-v1.2.3+compatible")
	t.NoError(err)

	b, err := json.Marshal(ht)
	t.NoError(err)

	var uht HintedString
	t.NoError(json.Unmarshal(b, &uht))

	t.Equal(ht.String(), uht.String())
}

func (t *testHintedString) TestEncodeBSON() {
	ht, err := ParseHintedString("showme~findme-v1.2.3+compatible")
	t.NoError(err)

	b, err := bson.Marshal(map[string]interface{}{
		"ht": ht,
	})
	t.NoError(err)

	var m map[string]HintedString
	t.NoError(bson.Unmarshal(b, &m))

	t.Equal(ht.String(), m["ht"].String())
}

func TestHintedString(t *testing.T) {
	suite.Run(t, new(testHintedString))
}
