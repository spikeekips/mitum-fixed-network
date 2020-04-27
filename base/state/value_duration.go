package state

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	DurationValueType = hint.MustNewType(0x12, 0x05, "state-duration-value")
	DurationValueHint = hint.MustHint(DurationValueType, "0.0.1")
)

type DurationValue struct {
	h valuehash.Hash
	v time.Duration
}

func NewDurationValue(d time.Duration) (DurationValue, error) {
	return DurationValue{}.set(d)
}

func (dv DurationValue) set(d time.Duration) (DurationValue, error) {
	dv.v = d
	dv.h = valuehash.NewSHA256(dv.Bytes())

	return dv, nil
}

func (dv DurationValue) IsValid([]byte) error {
	if err := dv.h.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (dv DurationValue) Bytes() []byte {
	return util.Int64ToBytes(dv.v.Nanoseconds())
}

func (dv DurationValue) Hint() hint.Hint {
	return DurationValueHint
}

func (dv DurationValue) Equal(v Value) bool {
	return dv.Hash().Equal(v.Hash())
}

func (dv DurationValue) Hash() valuehash.Hash {
	return dv.h
}

func (dv DurationValue) Interface() interface{} {
	return dv.v
}

func (dv DurationValue) Set(v interface{}) (Value, error) {
	d, ok := v.(time.Duration)
	if !ok {
		return nil, xerrors.Errorf("not time.Duration: %T", v)
	}

	return dv.set(d)
}
