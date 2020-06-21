package contestlib

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookup(t *testing.T) {
	cases := []struct {
		name       string
		jsonstring string
		key        string
		expected   interface{}
		found      bool
	}{
		{
			name:       "valid key; starts with digit",
			jsonstring: `{"0a":"A","b":1,"c":"C"}`,
			key:        "0a",
			expected:   "A",
			found:      true,
		},
		{
			name:       "bad key; empty key .",
			jsonstring: `{"a":"A","b":1,"c":"C"}`,
			key:        "",
			expected:   "",
			found:      false,
		},
		{
			name:       "bad key; ends with .",
			jsonstring: `{"a":"A","b":1,"c":"C"}`,
			key:        "a.",
			expected:   "",
			found:      false,
		},
		{
			name:       "simple: lookup, string #0",
			jsonstring: `{"a":"A","b":1,"c":"C"}`,
			key:        "a",
			expected:   "A",
			found:      true,
		},
		{
			name:       "simple: string #1",
			jsonstring: `{"a":"A","b":1,"c":"C"}`,
			key:        "c",
			expected:   "C",
			found:      true,
		},
		{
			name:       "simple: int",
			jsonstring: `{"a":"A","b":1,"c":"C"}`,
			key:        "b",
			expected:   float64(1),
			found:      true,
		},
		{
			name:       "simple: not found",
			jsonstring: `{"a":"A","b":1,"c":"C"}`,
			key:        "k",
			found:      false,
		},
		{
			name:       "nested: not found, #0",
			jsonstring: `{"a":"A","b":1,"c":{"d":"D","e":2,"f":"F"}}`,
			key:        "a.d",
			found:      false,
		},
		{
			name:       "nested: #1",
			jsonstring: `{"a":"A","b":1,"c":{"d":"D","e":2,"f":"F"}}`,
			key:        "c.d",
			expected:   "D",
			found:      true,
		},
		{
			name:       "nested: not found, #2",
			jsonstring: `{"a":"A","b":1,"c":{"d":"D","e":2,"f":"F"}}`,
			key:        "c.d.k",
			found:      false,
		},
		{
			name:       "nested: #3",
			jsonstring: `{"a":"A","b":1,"c":{"d":"D","e":2,"f":"F","g":{"h":"H","i":3,"j":"J"}}}`,
			key:        "c.g.i",
			expected:   float64(3),
			found:      true,
		},
		{
			name:       "nested: #3",
			jsonstring: `{"a":"A","b":1,"c":{"d":"D","e":2,"f":"F","g":{"h":"H","i":3,"j":"J"}}}`,
			key:        "c.g",
			found:      true,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func(*testing.T) {
				o := map[string]interface{}{}
				if err := json.Unmarshal([]byte(c.jsonstring), &o); err != nil {
					assert.NoError(t, err)
					return
				}

				result, found := Lookup(o, c.key)

				assert.Equal(t, c.found, found, "%d: %v; %v != %v", i, c.name, c.expected, result)
				if !c.found {
					return
				}

				if c.expected != nil {
					assert.Equal(t, c.expected, result, "%d: %v; %v != %v", i, c.name, c.expected, result)
				}
			},
		)
	}
}
