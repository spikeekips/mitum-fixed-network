package util

import (
	"math/big"

	"github.com/spikeekips/mitum/errors"
)

var (
	InvalidUnsignedIntError = errors.NewError("invalid UnsignedInt")
)

var (
	ZeroInt *big.Int = big.NewInt(0)
)

type UnsignedBigInt struct {
	*big.Int
}

func (us *UnsignedBigInt) IsValid() error {
	if ZeroInt.Cmp(us.BigInt()) > 0 {
		return InvalidUnsignedIntError.Wrapf("int=%v", us)
	}

	return nil
}

func (us UnsignedBigInt) BigInt() *big.Int {
	return us.Int
}

func NewUnsignedIntFromString(s string) (UnsignedBigInt, error) {
	i, ok := big.NewInt(0).SetString(s, 10)
	if !ok {
		return UnsignedBigInt{}, InvalidUnsignedIntError.Wrapf("string=%s", s)
	}

	us := UnsignedBigInt{Int: i}

	return us, us.IsValid()
}

func NewUnsignedInt(i int64) (UnsignedBigInt, error) {
	us := UnsignedBigInt{Int: big.NewInt(i)}
	return us, us.IsValid()
}

func NewUnsignedIntFromBigInt(b *big.Int) (UnsignedBigInt, error) {
	us := UnsignedBigInt{Int: b}
	return us, us.IsValid()
}
