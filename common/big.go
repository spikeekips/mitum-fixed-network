package common

import (
	"encoding/json"
	"math/big"

	"golang.org/x/xerrors"
)

var (
	ZeroBigInt *big.Int = new(big.Int).SetInt64(0)
	ZeroBig    Big      = NewBigFromInt64(0)
)

type Big struct {
	big.Int
}

func NewBigFromInt64(i int64) Big {
	var a big.Int
	a.SetInt64(i)

	return Big{Int: a}
}

func NewBigFromUint64(i uint64) Big {
	var a big.Int
	a.SetUint64(i)

	return Big{Int: a}
}

func ParseBig(s string) (Big, error) {
	var a big.Int
	err := a.UnmarshalText([]byte(s))
	if err != nil {
		return Big{}, err
	}

	return Big{Int: a}, nil
}

func (a Big) MarshalBinary() ([]byte, error) {
	return []byte(a.String()), nil
}

func (a *Big) UnmarshalBinary(b []byte) error {
	p, err := ParseBig(string(b))
	if err != nil {
		return err
	}

	*a = p

	return nil
}

func (a Big) MarshalJSON() ([]byte, error) {
	return json.Marshal(&a.Int)
}

func (a Big) String() string {
	return (&a.Int).String()
}

func (a Big) Inc() Big {
	b, _ := a.AddOK(NewBigFromInt64(1))
	return b
}

func (a Big) Add(v interface{}) Big {
	b, _ := a.AddOK(v)
	return b
}

func (a Big) AddOK(v interface{}) (Big, bool) {
	n, err := FromValue(v)
	if err != nil {
		return Big{}, false
	}

	var b big.Int
	b.Add(&a.Int, &n.Int)
	return Big{Int: b}, true
}

func (a Big) Sub(v interface{}) Big {
	b, _ := a.SubOK(v)
	return b
}

func (a Big) SubOK(v interface{}) (Big, bool) {
	n, err := FromValue(v)
	if err != nil {
		return Big{}, false
	}

	switch a.Int.Cmp(&n.Int) {
	case 0:
		return Big{}, true
	}

	var b big.Int
	b.Sub(&a.Int, &n.Int)
	return Big{Int: b}, true
}

func (a Big) Dec() Big {
	b, _ := a.SubOK(NewBigFromInt64(1))
	return b
}

func (a Big) MulOK(v interface{}) (Big, bool) {
	n, err := FromValue(v)
	if err != nil {
		return Big{}, false
	}

	var b big.Int
	b.Mul(&a.Int, &n.Int)
	return Big{Int: b}, true
}

func (a Big) Div(v interface{}) Big {
	b, _ := a.DivOK(v)
	return b
}

func (a Big) DivOK(v interface{}) (Big, bool) {
	n, err := FromValue(v)
	if err != nil {
		return Big{}, false
	}

	if n.Int.Cmp(ZeroBigInt) == 0 {
		return Big{}, false
	}

	var b big.Int
	b.Div(&a.Int, &n.Int)
	return Big{Int: b}, true
}

func (a Big) Rem(v interface{}) Big {
	b, _ := a.RemOK(v)
	return b
}

func (a Big) RemOK(v interface{}) (Big, bool) {
	n, err := FromValue(v)
	if err != nil {
		return Big{}, false
	}

	if n.Int.Cmp(ZeroBigInt) == 0 {
		return ZeroBig, true
	}

	var b big.Int
	b.Rem(&a.Int, &n.Int)
	return Big{Int: b}, true
}

func (a Big) Mul(v interface{}) Big {
	b, _ := a.MulOK(v)
	return b
}

func (a Big) IsZero() bool {
	return a.Int.Cmp(ZeroBigInt) == 0
}

func (a Big) UnderZero() bool {
	return a.Int.Cmp(ZeroBigInt) < 0
}

func (a Big) Cmp(v interface{}) int {
	n, err := FromValue(v)
	if err != nil {
		return -1
	}

	return a.Int.Cmp(&n.Int)
}

func (a Big) Equal(v interface{}) bool {
	n, err := FromValue(v)
	if err != nil {
		return false
	}

	return a.Int.Cmp(&n.Int) == 0
}

func (a Big) Int64() int64 {
	b, _ := a.Int64Ok()
	return b
}

func (a Big) Int64Ok() (int64, bool) {
	return (&(a.Int)).Int64(), (&(a.Int)).IsInt64()
}

func (a Big) Uint64() uint64 {
	b, _ := a.Uint64Ok()
	return b
}

func (a Big) Uint64Ok() (uint64, bool) {
	return (&(a.Int)).Uint64(), (&(a.Int)).IsUint64()
}

func FromValue(v interface{}) (Big, error) {
	switch v.(type) {
	default:
		return Big{}, xerrors.Errorf("invalid value; type=%q", v)
	case Big:
		return v.(Big), nil
	case int, int8, int16, int32, int64:
		var a int64
		switch v.(type) {
		case int:
			a = int64(v.(int))
		case int8:
			a = int64(v.(int8))
		case int16:
			a = int64(v.(int16))
		case int32:
			a = int64(v.(int32))
		case int64:
			a = v.(int64)
		}

		return NewBigFromInt64(a), nil
	case uint, uint8, uint16, uint32, uint64:
		var a uint64
		switch v.(type) {
		case uint:
			a = uint64(v.(uint))
		case uint8:
			a = uint64(v.(uint8))
		case uint16:
			a = uint64(v.(uint16))
		case uint32:
			a = uint64(v.(uint32))
		case uint64:
			a = uint64(v.(uint64))
		}
		return NewBigFromUint64(a), nil
	}
}
