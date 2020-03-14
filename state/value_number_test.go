package state

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testStateNumberValue struct {
	suite.Suite
}

func (t *testStateNumberValue) TestCases() {
	cases := []struct {
		name string
		v    interface{}
		err  string
	}{
		{name: "int", v: 10},
		{name: "int8", v: int8(10)},
		{name: "int16", v: int16(10)},
		{name: "int32", v: int32(10)},
		{name: "int64", v: int64(10)},
		{name: "uint", v: 10},
		{name: "uint8", v: uint8(10)},
		{name: "uint16", v: uint16(10)},
		{name: "uint32", v: uint32(10)},
		{name: "uint64", v: uint64(10)},
		{name: "float64", v: float64(10)},
		{name: "string", v: "find me", err: "not number-like"},
		{name: "bytes", v: []byte("find me"), err: "not number-like"},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				isv, err := NewNumberValue(c.v)
				if len(c.err) > 0 {
					if err == nil {
						t.NoError(xerrors.Errorf("error expected, but got %v", c.err), "%d: %v; %v != %v", i, c.name, c.err, err)
						return
					}

					t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
					return
				}

				if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
					return
				}

				t.NotNil(isv)
				t.Equal(c.v, isv.Interface())
				t.Equal(reflect.TypeOf(c.v).Kind(), isv.Type())
			},
		)
	}
}

func TestStateNumberValue(t *testing.T) {
	suite.Run(t, new(testStateNumberValue))
}
