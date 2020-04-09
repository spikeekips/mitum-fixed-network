package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/xerrors"
)

func TestThreshold(t *testing.T) {
	cases := []struct {
		name     string
		total    uint
		percent  float64
		expected uint // expected Threshold.Threshold
		err      string
	}{
		{
			name:    "0 total",
			total:   10,
			percent: 67,
			err:     "0 total",
		},
		{
			name:    "0 percent: 0",
			total:   10,
			percent: 0,
			err:     "0 percent",
		},
		{
			name:    "0 percent: under 1",
			total:   10,
			percent: 0.5,
			err:     "0 percent",
		},
		{
			name:    "over percent",
			total:   10,
			percent: 100.5,
			err:     "over 100 percent",
		},
		{
			name:     "threshold #0",
			total:    10,
			percent:  50,
			expected: 5,
		},
		{
			name:     "ceiled #0",
			total:    10,
			percent:  55,
			expected: 6,
		},
		{
			name:     "ceiled #1",
			total:    10,
			percent:  51,
			expected: 6,
		},
		{
			name:     "ceiled #1",
			total:    10,
			percent:  99,
			expected: 10,
		},
		{
			name:     "ceiled #1",
			total:    10,
			percent:  67,
			expected: 7,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func(*testing.T) {
				thr, err := NewThreshold(c.total, c.percent)
				if len(c.err) > 0 {
					if err == nil {
						assert.Error(t, xerrors.Errorf("expected error: %s, but nothing happened", c.err), "%d: %v", i, c.name)
						return
					}
					assert.Contains(t, err.Error(), c.err, "%d: %v", i, c.name)
					return
				}

				assert.Equal(t, c.expected, thr.Threshold, "%d: %v; %v != %v", i, c.name, c.expected, thr.Threshold)
			},
		)
	}
}
