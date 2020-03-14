package state

import (
	"testing"

	"github.com/spikeekips/mitum/hint"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testStateSliceValue struct {
	suite.Suite
}

func (t *testStateSliceValue) TestNotAcceptableValue() {
	cases := []struct {
		name string
		v    interface{}
		err  string
	}{
		{name: "string", v: "show me", err: "not slice-like"},
		{name: "empty string", v: "", err: "not slice-like"},
		{name: "int", v: 10, err: "not slice-like"},
		{name: "[]int", v: []int{1, 2, 3}, err: "item not hint.Hinter"},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				_, err := NewSliceValue(c.v)
				if err == nil {
					t.NoError(xerrors.Errorf("error expected, but got %v", c.err), "%d: %v; %v != %v", i, c.name, c.err, err)
					return
				}

				t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
				return
			},
		)
	}
}

func (t *testStateSliceValue) TestNew() {
	var items []dummy
	for i := 0; i < 3; i++ {
		d := dummy{}
		d.v = i
		items = append(items, d)
	}

	sl, err := NewSliceValue(items)
	t.NoError(err)
	t.NotNil(sl.Hash())

	for i, h := range sl.Interface().([]hint.Hinter) {
		d := h.(dummy)
		t.Equal(items[i], d)
	}
}

func TestStateSliceValue(t *testing.T) {
	suite.Run(t, new(testStateSliceValue))
}
