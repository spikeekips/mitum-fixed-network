package state

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	DurationValueType   = hint.Type("state-duration-value")
	DurationValueHint   = hint.NewHint(DurationValueType, "v0.0.1")
	DurationValueHinter = DurationValue{BaseHinter: hint.NewBaseHinter(DurationValueHint)}
)

type DurationValue struct {
	hint.BaseHinter
	h valuehash.Hash
	v time.Duration
}

func NewDurationValue(d time.Duration) (DurationValue, error) {
	return DurationValue{BaseHinter: hint.NewBaseHinter(DurationValueHint)}.set(d)
}

func (dv DurationValue) set(d time.Duration) (DurationValue, error) {
	dv.v = d
	dv.h = valuehash.NewSHA256(util.Int64ToBytes(dv.v.Nanoseconds()))

	return dv, nil
}

func (dv DurationValue) IsValid([]byte) error {
	return isvalid.Check([]isvalid.IsValider{dv.BaseHinter, dv.h}, nil, false)
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
		return nil, errors.Errorf("not time.Duration: %T", v)
	}

	return dv.set(d)
}
