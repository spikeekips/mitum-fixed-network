package hint

import (
	"encoding/json"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

type testHint struct {
	suite.Suite
}

func (t *testHint) TestNew() {
	ht := NewHint("showme", "v1.2.3+compatible")
	t.NoError(ht.IsValid(nil))
	t.Equal("showme-v1.2.3+compatible", ht.String())
}

func (t *testType) TestParse() {
	cases := []struct {
		name     string
		s        string
		expected string
		err      string
	}{
		{name: "valid", s: "showme-v1.2.3+incompatible"},
		{name: "empty version #0", s: "showme", err: "invalid Hint format"},
		{name: "empty version #1", s: "showme-", err: "invalid Hint format"},
		{name: "inside v+hyphen", s: "sho-vwme-v1.2.3+incompatible", expected: "sho-vwme-v1.2.3+incompatible"},
		{name: "hyphen-v", s: "sho-v1.2.3wme-v1.2.3+incompatible", expected: "sho-v1.2.3wme-v1.2.3+incompatible"},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				ht, err := ParseHint(c.s)
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

				if len(c.expected) > 0 {
					t.Equal(c.expected, ht.String(), "%d: %v", i, c.name)
				}
			},
		)
	}
}

func (t *testType) TestCompatible() {
	cases := []struct {
		name string
		a    string
		b    string
		err  string
	}{
		{name: "upper patch version", a: "showme-v1.2.3", b: "showme-v1.2.4"},
		{name: "same patch version", a: "showme-v1.2.3", b: "showme-v1.2.3"},
		{name: "lower patch version", a: "showme-v1.2.3", b: "showme-v1.2.2"},
		{name: "upper minor version", a: "showme-v1.2.3", b: "showme-v1.3.4", err: "not compatible; lower minor version"},
		{name: "same minor version", a: "showme-v1.2.3", b: "showme-v1.2.9"},
		{name: "lower minor version", a: "showme-v1.2.3", b: "showme-v1.1.2"},
		{name: "upper major version", a: "showme-v1.2.3", b: "showme-v9.3.4", err: "not compatible; different major version"},
		{name: "same major version", a: "showme-v1.2.3", b: "showme-v10.2.9", err: "not compatible; different major version"},
		{name: "lower major version", a: "showme-v1.2.3", b: "showme-v2.1.2", err: "not compatible; different major version"},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				a, _ := ParseHint(c.a)
				b, _ := ParseHint(c.b)

				err := a.IsCompatible(b)

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

func (t *testType) TestEncodeJSON() {
	ht := NewHint("showme", "v1.2.3+compatible")
	b, err := json.Marshal(map[string]interface{}{
		"_hint": ht,
	})
	t.NoError(err)

	var m map[string]Hint
	t.NoError(json.Unmarshal(b, &m))

	uht := m["_hint"]
	t.True(ht.Equal(uht))
}

func (t *testType) TestEncodeBSON() {
	ht := NewHint("showme", "v1.2.3+compatible")
	b, err := bson.Marshal(map[string]interface{}{
		"_hint": ht,
	})
	t.NoError(err)

	var m map[string]Hint
	t.NoError(bson.Unmarshal(b, &m))

	uht := m["_hint"]
	t.True(ht.Equal(uht))
}

func TestHint(t *testing.T) {
	suite.Run(t, new(testHint))
}
