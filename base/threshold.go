package base

import (
	"math"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/isvalid"
)

type ThresholdRatio float64

func (tr ThresholdRatio) Float64() float64 {
	return float64(tr)
}

func (tr ThresholdRatio) IsValid([]byte) error {
	if tr < 1 {
		return isvalid.InvalidError.Errorf("0 ratio found: %v", tr)
	} else if tr > 100 {
		return isvalid.InvalidError.Errorf("over 100 ratio: %v", tr)
	}

	return nil
}

type Threshold struct {
	Total     uint           `json:"total" bson:"total"`
	Threshold uint           `json:"threshold" bson:"threshold"`
	Ratio     ThresholdRatio `json:"ratio" bson:"ratio"` // NOTE 67.0 ~ 100.0
}

func NewThreshold(total uint, ratio ThresholdRatio) (Threshold, error) {
	thr := Threshold{
		Total:     total,
		Threshold: uint(math.Ceil(float64(total) * (ratio.Float64() / 100))),
		Ratio:     ratio,
	}

	return thr, thr.IsValid(nil)
}

func MustNewThreshold(total uint, ratio ThresholdRatio) Threshold {
	thr, err := NewThreshold(total, ratio)
	if err != nil {
		panic(err)
	}

	return thr
}

func (thr Threshold) Bytes() []byte {
	return util.ConcatBytesSlice(
		util.UintToBytes(thr.Total),
		util.Float64ToBytes(thr.Ratio.Float64()),
	)
}

func (thr Threshold) String() string {
	b, _ := jsonenc.Marshal(thr)
	return string(b)
}

func (thr Threshold) Equal(b Threshold) bool {
	if thr.Total != b.Total {
		return false
	}
	if thr.Ratio != b.Ratio {
		return false
	}
	if thr.Threshold != b.Threshold {
		return false
	}

	return true
}

func (thr Threshold) IsValid([]byte) error {
	if err := thr.Ratio.IsValid(nil); err != nil {
		return err
	}
	if thr.Total < 1 {
		return errors.Errorf("zero total found")
	}
	if thr.Threshold > thr.Total {
		return isvalid.InvalidError.Errorf("Threshold over Total: Threshold=%v Total=%v", thr.Threshold, thr.Total)
	}

	return nil
}
