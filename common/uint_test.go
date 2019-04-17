package common

import (
	"math"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testUint struct {
	suite.Suite
}

func (t *testUint) TestAdd() {
	t.Equal(Uint(21), Uint(10)+Uint(11))

	{ // safe
		returned, ok := Uint(10).AddOK(Uint(11))
		t.True(ok)
		t.Equal(Uint(21), returned)
	}

	{ // overflow
		returned, ok := Uint(10).AddOK(Uint(math.MaxUint64))
		t.False(ok)
		t.Equal(Uint(0), returned)
	}

	{ // max + 0
		returned, ok := Uint(math.MaxUint64).AddOK(Uint(0))
		t.True(ok)
		t.Equal(Uint(math.MaxUint64), returned)
	}

	{ // overflow
		returned, ok := Uint(math.MaxUint64).AddOK(Uint(1))
		t.False(ok)
		t.Equal(Uint(0), returned)
	}
}

func (t *testUint) TestSub() {
	t.Equal(Uint(1), Uint(11)-Uint(10))

	{ // safe
		returned, ok := Uint(11).SubOK(Uint(10))
		t.True(ok)
		t.Equal(Uint(1), returned)
	}

	{ // overflow
		returned, ok := Uint(10).SubOK(Uint(math.MaxUint64))
		t.False(ok)
		t.Equal(Uint(0), returned)
	}

	{ // max - 0
		returned, ok := Uint(math.MaxUint64).SubOK(Uint(0))
		t.True(ok)
		t.Equal(Uint(math.MaxUint64), returned)
	}

	{ // max - max
		returned, ok := Uint(math.MaxUint64).SubOK(Uint(math.MaxUint64))
		t.True(ok)
		t.Equal(Uint(0), returned)
	}
}

func (t *testUint) TestMul() {
	t.Equal(Uint(12), Uint(3)*Uint(4))

	{ // safe
		returned, ok := Uint(11).MulOK(Uint(10))
		t.True(ok)
		t.Equal(Uint(110), returned)
	}

	{ // overflow
		returned, ok := Uint(math.MaxUint64 - 3).MulOK(Uint(2))
		t.False(ok)
		t.Equal(Uint(0), returned)
	}

	{ // max * 1
		returned, ok := Uint(math.MaxUint64).MulOK(Uint(1))
		t.True(ok)
		t.Equal(Uint(math.MaxUint64), returned)
	}
}

func (t *testUint) TestDiv() {
	t.Equal(Uint(0), Uint(3)/Uint(4))

	{ // safe
		returned, ok := Uint(11).DivOK(Uint(10))
		t.True(ok)
		t.Equal(Uint(1), returned)
	}
}

func TestUint(t *testing.T) {
	suite.Run(t, new(testUint))
}
