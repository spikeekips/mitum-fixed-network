package isaac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestThreshold(t *testing.T) {
	cases := []struct {
		name     string
		total    uint
		percent  float64
		expected uint
		err      string
	}{
		{
			name:  "basic",
			total: 10, percent: 66,
			expected: 7,
		},
		{
			name:  "over 100",
			total: 10, percent: 166,
			err: "is over 100",
		},
		{
			name:  "100 percent",
			total: 10, percent: 100,
			expected: 10,
		},
		{
			name:  "100 percent",
			total: 33, percent: 33.45,
			expected: 12,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func(*testing.T) {
				result, err := NewThreshold(c.total, c.percent)
				if len(c.err) > 0 {
					assert.Contains(t, err.Error(), c.err)
				} else {
					assert.Equal(t, c.expected, result.base[1], "%d: %v; %v != %v", i, c.name, c.expected, result.base[1])
				}
			},
		)
	}
}
