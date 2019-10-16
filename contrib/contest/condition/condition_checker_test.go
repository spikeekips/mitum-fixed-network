package condition

import (
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
		{name: "simple: matched", query: "a=1", o: `{"a": 1}`, expected: true},
		{name: "simple: not matched", query: "a=1", o: `{"a": 2}`, expected: false},
		{name: "simple: greater than #0", query: "a>0", o: `{"a": 1}`, expected: true},
		{name: "simple: greater than #1", query: "a>=1", o: `{"a": 1}`, expected: true},
		{name: "simple: not greater than", query: "a>2", o: `{"a": 2}`, expected: false},
		{name: "in: included", query: "a in (1,2,3)", o: `{"a": 1}`, expected: true},
		{name: "in: not included", query: "a in (1,2,3)", o: `{"a": 4}`, expected: false},
		{name: "not in: included", query: "a not in (1,2,3)", o: `{"a": 1}`, expected: false},
		{name: "not in: not included", query: "a not in (1,2,3)", o: `{"a": 4}`, expected: true},
		{name: "regexp: matched", query: `a regexp "^1$"`, o: `{"a": 1}`, expected: true},
		{name: "complex: #0", query: `b.d regexp "^showme$"`, o: `{"a": 1, "b": {"c": 2, "d": "showme"}}`, expected: true},
		{name: "and: matched", query: `a=1 AND b.c=2`, o: `{"a": 1, "b": {"c": 2, "d": "showme"}}`, expected: true},
		{name: "and: not matched #0", query: `a=2 AND b.c=2`, o: `{"a": 1, "b": {"c": 2, "d": "showme"}}`, expected: false},
		{name: "and: not matched #1", query: `a=1 AND b.c=3`, o: `{"a": 1, "b": {"c": 2, "d": "showme"}}`, expected: false},
		{name: "or: matched", query: `a=1 OR b.c=2`, o: `{"a": 1, "b": {"c": 2, "d": "showme"}}`, expected: true},
		{name: "or: not matched", query: `a=2 OR b.c=3`, o: `{"a": 1, "b": {"c": 2, "d": "showme"}}`, expected: false},
		{name: "nested: matched", query: `(a=1 OR b.c=3) AND (a=2 OR b.d="showme")`, o: `{"a": 1, "b": {"c": 2, "d": "showme"}}`, expected: true},
		{name: "null: matched", query: `a=null`, o: `{"a": null}`, expected: true},
		{name: "null: string matched", query: `a=""`, o: `{"a": null}`, expected: false},
		{name: "null: int matched", query: `a=1`, o: `{"a": null}`, expected: false},
		{name: "null: float matched #0", query: `a=1.1`, o: `{"a": null}`, expected: false},
		{name: "null: float matched #1", query: `a>1.1`, o: `{"a": null}`, expected: false},
		{name: "null: float matched #2", query: `a=null`, o: `{"a": 1}`, expected: false},
		{name: "boolean: true", query: `a=true`, o: `{"a": true}`, expected: true},
		{name: "boolean: false", query: `a=false`, o: `{"a": false}`, expected: true},
		{name: "boolean: false; uppercase", query: `a=False`, o: `{"a": false}`, expected: true},
		{name: "boolean: string", query: `a=true`, o: `{"a": "showme"}`, expected: true},
		{name: "boolean: empty string", query: `a=true`, o: `{"a": ""}`, expected: false},
		{name: "boolean: int 1", query: `a=true`, o: `{"a": 1}`, expected: true},
		{name: "boolean: int 0", query: `a=true`, o: `{"a": 0}`, expected: false},
		{name: "boolean: list", query: `a=true`, o: `{"a": [1,2]}`, expected: true},
		{name: "boolean: empty list", query: `a=true`, o: `{"a": []}`, expected: false},
		{name: "boolean: empty list", query: `a=false`, o: `{"a": []}`, expected: true},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func(*testing.T) {
				o, err := NewLogItem([]byte(c.o))
				if err != nil {
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
