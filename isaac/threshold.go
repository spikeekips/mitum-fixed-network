package isaac

import (
	"encoding/json"
	"math"
	"sync"

	"golang.org/x/xerrors"
)

type Threshold struct {
	sync.RWMutex
	base      [2]uint // [2]uint{total, threshold}
	threshold map[Stage][2]uint
}

func NewThreshold(baseTotal uint, basePercent float64) (*Threshold, error) {
	th, err := calculateThreshold(baseTotal, basePercent)
	if err != nil {
		return nil, err
	}

	return &Threshold{
		base:      [2]uint{baseTotal, th},
		threshold: map[Stage][2]uint{},
	}, nil
}

func (tr *Threshold) Get(stage Stage) (uint, uint) {
	tr.RLock()
	defer tr.RUnlock()

	t, found := tr.threshold[stage]
	if found {
		return t[0], t[1]
	}

	return tr.base[0], tr.base[1]
}

func (tr *Threshold) SetBase(baseTotal uint, basePercent float64) error {
	tr.Lock()
	defer tr.Unlock()

	th, err := calculateThreshold(baseTotal, basePercent)
	if err != nil {
		return err
	}

	tr.base = [2]uint{baseTotal, th}

	return nil
}

func (tr *Threshold) Set(stage Stage, total uint, percent float64) error {
	tr.Lock()
	defer tr.Unlock()

	th, err := calculateThreshold(total, percent)
	if err != nil {
		return err
	}

	tr.threshold[stage] = [2]uint{total, th}

	return nil
}

func (tr *Threshold) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"base":      tr.base,
		"threshold": tr.threshold,
	})
}

func (tr *Threshold) String() string {
	b, _ := json.Marshal(tr) // nolint
	return string(b)
}

func calculateThreshold(total uint, percent float64) (uint, error) {
	if percent > 100 {
		return 0, xerrors.Errorf("basePercent is over 100; %v", percent)
	}

	return uint(math.Ceil(float64(total) * (percent / 100))), nil
}
