package base

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestThreshold(t *testing.T) {
	cases := []struct {
		name     string
		total    uint
		ratio    float64
		expected uint // expected Threshold.Threshold
		err      string
	}{
		{
			name:  "0 total",
			total: 10,
			ratio: 67,
			err:   "0 total",
		},
		{
			name:  "0 ratio: 0",
			total: 10,
			ratio: 0,
			err:   "0 ratio",
		},
		{
			name:  "0 ratio: under 1",
			total: 10,
			ratio: 0.5,
			err:   "0 ratio",
		},
		{
			name:  "over ratio",
			total: 10,
			ratio: 100.5,
			err:   "over 100 ratio",
		},
		{
			name:     "threshold #0",
			total:    10,
			ratio:    50,
			expected: 5,
		},
		{
			name:     "ceiled #0",
			total:    10,
			ratio:    55,
			expected: 6,
		},
		{
			name:     "ceiled #1",
			total:    10,
			ratio:    51,
			expected: 6,
		},
		{
			name:     "ceiled #1",
			total:    10,
			ratio:    99,
			expected: 10,
		},
		{
			name:     "ceiled #1",
			total:    10,
			ratio:    67,
			expected: 7,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func(*testing.T) {
				thr, err := NewThreshold(c.total, ThresholdRatio(c.ratio))
				if len(c.err) > 0 {
					if err == nil {
						assert.Error(t, errors.Errorf("expected error: %s, but nothing happened", c.err), "%d: %v", i, c.name)
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

func TestNumberOfFaultyNodes(t *testing.T) {
	cases := []struct {
		name      string
		n         uint
		threshold float64
		expected  int
	}{
		{
			name:      "3, 67",
			n:         3,
			threshold: 67,
			expected:  0,
		},
		{
			name:      "3, 60",
			n:         3,
			threshold: 60,
			expected:  1,
		},
		{
			name:      "0, 60",
			n:         0,
			threshold: 60,
			expected:  0,
		},
		{
			name:      "10, 60",
			n:         10,
			threshold: 60,
			expected:  4,
		},
		{
			name:      "10, 61",
			n:         10,
			threshold: 61,
			expected:  3,
		},
		{
			name:      "10, 100",
			n:         10,
			threshold: 100,
			expected:  0,
		},
		{
			name:      "33, 200",
			n:         33,
			threshold: 200,
			expected:  0,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func(*testing.T) {
				f := NumberOfFaultyNodes(c.n, c.threshold)
				assert.Equal(t, c.expected, f, "%d: %v; %v != %v", i, c.name, c.expected, f)
			},
		)
	}
}
