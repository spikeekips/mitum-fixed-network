package common

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testAmount struct {
	suite.Suite
}

func (t *testAmount) TestAdd() {
	{ // big.Int: add overflow, but ok
		var a, b, c big.Int
		a.SetUint64(10)
		b.SetUint64(math.MaxUint64)

		t.Equal(1, c.Add(&a, &b).Cmp(&b))
	}

	{
		a := NewAmount(10)
		b := NewAmount(math.MaxUint64)

		c, ok := a.AddOK(b)
		t.True(ok)

		t.Equal("10", c.Int.Sub(&c.Int, &b.Int).String())
	}

	{
		a := NewAmount(math.MaxUint64)
		b := NewAmount(math.MaxUint64)

		c, ok := a.AddOK(b)
		t.True(ok)
		t.Equal("36893488147419103230", c.Int.String())

		t.Equal(a.Int, *c.Sub(&c.Int, &b.Int))
		t.Equal("18446744073709551615", b.Int.String())
	}
}

func (t *testAmount) TestSub() {
	{
		a := NewAmount(10)
		b := NewAmount(math.MaxUint64)

		c, ok := b.SubOK(a)
		t.True(ok)
		t.Equal("18446744073709551605", c.Int.String())
	}

	{
		a := NewAmount(10)
		b := NewAmount(math.MaxUint64)

		c, ok := a.SubOK(b)
		t.False(ok)
		t.Equal("0", c.Int.String())
	}

	{
		a := NewAmount(math.MaxUint64)
		b := NewAmount(math.MaxUint64)
		c, _ := a.AddOK(b)

		d, ok := c.SubOK(a)
		t.True(ok)
		t.Equal("18446744073709551615", d.Int.String())
	}
}

func (t *testAmount) TestMul() {
	{
		a := NewAmount(10)
		b := NewAmount(math.MaxUint64)

		c, ok := b.MulOK(a)
		t.True(ok)
		t.Equal("184467440737095516150", c.Int.String())
	}

	{
		a := NewAmount(math.MaxUint64)
		b := NewAmount(math.MaxUint64)
		c, _ := a.AddOK(b)

		d, ok := c.MulOK(a)
		t.True(ok)
		t.Equal("680564733841876926852962238568698216450", d.Int.String())
	}
}

func (t *testAmount) TestDiv() {
	{
		a := NewAmount(10)
		b := NewAmount(math.MaxUint64)

		c, ok := b.DivOK(a)
		t.True(ok)
		t.Equal("1844674407370955161", c.Int.String())
	}

	{ // divizion zero
		a := NewAmount(10)
		b := NewAmount(0)

		c, ok := b.DivOK(a)
		t.True(ok)
		t.Equal("0", c.Int.String())
	}

	{ // zero divizion
		a := NewAmount(10)
		b := NewAmount(0)

		c, ok := a.DivOK(b)
		t.False(ok)
		t.Equal("0", c.Int.String())
	}
}

func TestAmount(t *testing.T) {
	suite.Run(t, new(testAmount))
}
