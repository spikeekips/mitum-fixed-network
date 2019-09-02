package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCondition(t *testing.T) {
	cases := []struct {
		name     string
		where    string
		expected string
		err      string
	}{
		{
			name:     "simple: int",
			where:    "a=2",
			expected: "(a = [2])",
		},
		{
			name:     "simple: string val",
			where:    `a="showme"`,
			expected: "(a = [showme])",
		},
		{
			name:     "simple: negative int",
			where:    "a=-2",
			expected: "(a = [-2])",
		},
		{
			name:     "simple: float",
			where:    "a=3.141592",
			expected: "(a = [3.141592])",
		},
		{
			name:     "simple: dot connected field",
			where:    "a.b.c.d.e = 2",
			expected: "(a.b.c.d.e = [2])",
		},
		{
			name:     "simple: a>=2",
			where:    "a>=2",
			expected: "(a >= [2])",
		},
		{
			name:     "simple: equal a = 3",
			where:    "a=3",
			expected: "(a = [3])",
		},
		{
			name:     "simple: lessthan a < 3",
			where:    "a<3",
			expected: "(a < [3])",
		},
		{
			name:     "simple: greaterthan a > 3",
			where:    "a>3",
			expected: "(a > [3])",
		},
		{
			name:     "simple: lessequal a <= 3",
			where:    "a<=3",
			expected: "(a <= [3])",
		},
		{
			name:     "simple: greaterequal a >= 3",
			where:    "a>=3",
			expected: "(a >= [3])",
		},
		{
			name:     "simple: notequal a != 3",
			where:    "a!=3",
			expected: "(a != [3])",
		},
		{
			name:     "simple: in a in 3",
			where:    "a in (3, 4, 5, 6)",
			expected: "(a in [3,4,5,6])",
		},
		{
			name:     "simple: notin a not in 3",
			where:    "a not in (3, 4, 5, 6)",
			expected: "(a not in [3,4,5,6])",
		},
		{
			name:     "simple: in a in 3",
			where:    "a in (3, 4, 5, 6)",
			expected: "(a in [3,4,5,6])",
		},
		{
			name:     "simple: notin a not in 3",
			where:    "a not in (3, 4, 5, 6)",
			expected: "(a not in [3,4,5,6])",
		},
		{
			name:     "simple: regexp a regexp 3",
			where:    `a regexp "foo.*"`,
			expected: "(a regexp [foo.*])",
		},
		{
			name:     "simple: notregexp a not regexp 3",
			where:    `a not regexp "foo.*"`,
			expected: "(a not regexp [foo.*])",
		},
		{
			name:  "simple: bad regexp expression",
			where: `a not regexp "foo(.*"`,
			err:   "error parsing regexp",
		},
		{
			name:     "joint: and with 2 comparison",
			where:    `a = 1 and b = 2`,
			expected: "(and:(a = [1]), (b = [2]))",
		},
		{
			name:     "joint: and with 3 comparison",
			where:    `a = 1 and b = 2 and c = 3`,
			expected: "(and:(a = [1]), (b = [2]), (c = [3]))",
		},
		{
			name:     "joint: or with 2 comparison",
			where:    `a = 1 or b = 2`,
			expected: "(or:(a = [1]), (b = [2]))",
		},
		{
			name:     "joint: or with 3 comparison",
			where:    `a = 1 or b = 2 or c = 3`,
			expected: "(or:(a = [1]), (b = [2]), (c = [3]))",
		},
		{
			name:     "joint: and first, complex with 3 comparison",
			where:    `a = 1 and b = 2 or c = 3`,
			expected: "(or:(and:(a = [1]), (b = [2])), (c = [3]))",
		},
		{
			name:     "joint: or first, complex with 3 comparison",
			where:    `(a = 1 or b = 2) and c = 3`,
			expected: "(and:(or:(a = [1]), (b = [2])), (c = [3]))",
		},
		{
			name:     "joint: complex #0",
			where:    `(a > 1 or b < 2) and (c >= 3 and d <= 4) or (e != 5 and f not in (6, 7))`,
			expected: "(or:(and:(or:(a > [1]), (b < [2])), (and:(c >= [3]), (d <= [4]))), (and:(e != [5]), (f not in [6,7])))",
		},
		{
			name:     "joint: complex #1",
			where:    `(a.x.y.z > 1 or b < 2) and (c.o.p.q.r >= 3 and d.s.t.u <= 4) or (e.v.w != 5 and f.m.n not in (6, 7))`,
			expected: "(or:(and:(or:(a.x.y.z > [1]), (b < [2])), (and:(c.o.p.q.r >= [3]), (d.s.t.u <= [4]))), (and:(e.v.w != [5]), (f.m.n not in [6,7])))",
		},
		{
			name:     "null: #0",
			where:    `a = null`,
			expected: "(a = [])",
		},
	}

	cp := NewConditionParser()
	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func(*testing.T) {
				result, err := cp.Parse(c.where)
				if len(c.err) > 0 {
					errString := ""
					if err != nil {
						errString = err.Error()
					}

					assert.Contains(t, errString, c.err, "%d: %v; %v != %v", i, c.name, c.expected, result)
					return
				} else if err != nil {
					assert.NoError(t, err)
					return
				}

				assert.Equal(t, c.expected, result.String(), "%d: %v; %v != %v", i, c.name, c.expected, result)
			},
		)
	}
}
