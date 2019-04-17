package common

import (
	"encoding/json"
	"math/big"
)

var (
	ZeroBigInt *big.Int = new(big.Int).SetInt64(0)
)

type Amount struct {
	big.Int
}

func NewAmount(i uint64) Amount {
	var a big.Int
	a.SetUint64(i)

	return Amount{Int: a}
}

func (a Amount) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Bytes())
}

func (a *Amount) UnmarshalJSON(b []byte) error {
	var n []byte
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}

	i := new(big.Int).SetBytes(n)

	*a = Amount{Int: *i}
	return nil
}

func (a Amount) AddOK(n Amount) (Amount, bool) {
	var b big.Int
	b.Add(&a.Int, &n.Int)
	return Amount{Int: b}, true
}

func (a Amount) SubOK(n Amount) (Amount, bool) {
	switch a.Int.Cmp(&n.Int) {
	case -1:
		return Amount{}, false
	case 0:
		return Amount{}, true
	}

	var b big.Int
	b.Sub(&a.Int, &n.Int)
	return Amount{Int: b}, true
}

func (a Amount) MulOK(n Amount) (Amount, bool) {
	var b big.Int
	b.Mul(&a.Int, &n.Int)
	return Amount{Int: b}, true
}

func (a Amount) DivOK(n Amount) (Amount, bool) {
	if n.Int.Cmp(ZeroBigInt) == 0 {
		return Amount{}, false
	}

	var b big.Int
	b.Div(&a.Int, &n.Int)
	return Amount{Int: b}, true
}

func (a Amount) Mul(n Amount) Amount {
	b, _ := a.MulOK(n)
	return b
}

func (a Amount) IsZero(n Amount) bool {
	return n.Int.Cmp(ZeroBigInt) == 0
}
