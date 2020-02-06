package isaac

import (
	"math"

	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
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

func (thr Threshold) String() string {
	b, _ := util.JSONMarshal(thr)
	return string(b)
}

func (thr Threshold) IsValid([]byte) error {
	if thr.Total < 1 {
		return xerrors.Errorf("0 total")
	}
	if thr.Percent < 1 {
		return InvalidError.Wrapf("0 percent: %v", thr.Percent)
	} else if thr.Percent > 100 {
		return InvalidError.Wrapf("over 100 percent: %v", thr.Percent)
	}
	if thr.Threshold > thr.Total {
		return InvalidError.Wrapf("Threshold over Total: Threshold=%v Total=%v", thr.Threshold, thr.Total)
	}

	return nil
}
