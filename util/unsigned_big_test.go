package util

import (
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"golang.org/x/xerrors"
)

type testUnsignedInt struct {
	suite.Suite
}

func (t *testUnsignedInt) TestNew() {
	us, err := NewUnsignedInt(10)
	t.NoError(err)

	t.Equal("10", us.Text(10))

	osi, err := NewUnsignedIntFromBigInt(
		big.NewInt(0).Mul(
			big.NewInt(2),
			big.NewInt(0).SetUint64(math.MaxUint64),
		),
	)
	t.NoError(err)

	t.Equal(big.NewInt(0).SetUint64(math.MaxUint64), osi.Div(osi.BigInt(), big.NewInt(2)))
}

func (t *testUnsignedInt) TestUnderZero() {
	{
		_, err := NewUnsignedInt(-10)
		t.True(xerrors.Is(err, InvalidUnsignedIntError))
	}

	{
		_, err := NewUnsignedIntFromString("-10")
		t.True(xerrors.Is(err, InvalidUnsignedIntError))
	}

	{
		_, err := NewUnsignedIntFromBigInt(big.NewInt(-10))
		t.True(xerrors.Is(err, InvalidUnsignedIntError))
	}
}

func TestUnsignedInt(t *testing.T) {
	defer goleak.VerifyNone(t)

	suite.Run(t, new(testUnsignedInt))
}
