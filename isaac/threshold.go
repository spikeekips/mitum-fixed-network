package isaac

import (
	"math"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/util"
)

type Threshold struct {
	Total     uint    `json:"total"`
	Threshold uint    `json:"threshold"`
	Percent   float64 `json:"percent"` // NOTE 67.0 ~ 100.0
}

func NewThreshold(total uint, percent float64) (Threshold, error) {
	thr := Threshold{
		Total:     total,
		Threshold: uint(math.Ceil(float64(total) * (percent / 100))),
		Percent:   percent,
	}

	return thr, thr.IsValid(nil)
}

func MustNewThreshold(total uint, percent float64) Threshold {
	thr, err := NewThreshold(total, percent)
	if err != nil {
		panic(err)
	}

	return thr
}

func (thr Threshold) Bytes() []byte {
	return util.ConcatBytesSlice(
		util.UintToBytes(thr.Total),
		util.Float64ToBytes(thr.Percent),
	)
}

func (thr Threshold) String() string {
	b, _ := util.JSONMarshal(thr)
	return string(b)
}

func (thr Threshold) Equal(b Threshold) bool {
	if thr.Total != b.Total {
		return false
	}
	if thr.Percent != b.Percent {
		return false
	}
	if thr.Threshold != b.Threshold {
		return false
	}

	return true
}

func (thr Threshold) IsValid([]byte) error {
	if thr.Total < 1 {
		return xerrors.Errorf("zero total found")
	}
	if thr.Percent < 1 {
		return isvalid.InvalidError.Wrapf("0 percent found: %v", thr.Percent)
	} else if thr.Percent > 100 {
		return isvalid.InvalidError.Wrapf("over 100 percent: %v", thr.Percent)
	}
	if thr.Threshold > thr.Total {
		return isvalid.InvalidError.Wrapf("Threshold over Total: Threshold=%v Total=%v", thr.Threshold, thr.Total)
	}

	return nil
}
