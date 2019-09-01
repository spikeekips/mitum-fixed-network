package main

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConditionChecker(t *testing.T) {
	cases := []struct {
		name     string
		query    string
		o        string
		expected bool
	}{
		{
			name:     "simple: matched",
			query:    "a=1",
			o:        `{"a": 1}`,
			expected: true,
		},
		{
			name:     "simple: not matched",
			query:    "a=1",
			o:        `{"a": 2}`,
			expected: false,
		},
		{
			name:     "simple: greater than #0",
			query:    "a>0",
			o:        `{"a": 1}`,
			expected: true,
		},
		{
			name:     "simple: greater than #1",
			query:    "a>=1",
			o:        `{"a": 1}`,
			expected: true,
		},
		{
			name:     "simple: not greater than",
			query:    "a>2",
			o:        `{"a": 2}`,
			expected: false,
		},
		{
			name:     "in: included",
			query:    "a in (1,2,3)",
			o:        `{"a": 1}`,
			expected: true,
		},
		{
			name:     "in: not included",
			query:    "a in (1,2,3)",
			o:        `{"a": 4}`,
			expected: false,
		},
		{
			name:     "not in: included",
			query:    "a not in (1,2,3)",
			o:        `{"a": 1}`,
			expected: false,
		},
		{
			name:     "not in: not included",
			query:    "a not in (1,2,3)",
			o:        `{"a": 4}`,
			expected: true,
		},
		{
			name:     "regexp: matched",
			query:    `a regexp "^1$"`,
			o:        `{"a": 1}`,
			expected: true,
		},
		{
			name:     "complex: #0",
			query:    `b.d regexp "^showme$"`,
			o:        `{"a": 1, "b": {"c": 2, "d": "showme"}}`,
			expected: true,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func(*testing.T) {
				o := map[string]interface{}{}
				if err := json.Unmarshal([]byte(c.o), &o); err != nil {
					assert.NoError(t, err)
					return
				}

				cc, err := NewConditionChecker(c.query)
				if err != nil {
					assert.NoError(t, err)
					return
				}

				result := cc.Check(o)

				assert.Equal(t, c.expected, result,
					"%d: %v; %v; %v; %v != %v", i, c.name, c.query, c.o, c.expected, result,
				)
			},
		)
	}
}
