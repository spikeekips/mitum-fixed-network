package isaac

import (
	"encoding/json"
	"math"
	"sync"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"
)

type Threshold struct {
	sync.RWMutex
	base      [3]uint // [2]uint{total, threshold}
	threshold map[Stage][3]uint
}

func NewThreshold(baseTotal uint, basePercent float64) (*Threshold, error) {
	th, err := calculateThreshold(baseTotal, basePercent)
	if err != nil {
		return nil, err
	}

	return &Threshold{
		base:      [3]uint{baseTotal, th, uint(basePercent * 100)},
		threshold: map[Stage][3]uint{},
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

	tr.base = [3]uint{baseTotal, th, uint(basePercent * 100)}

	return nil
}

func (tr *Threshold) Set(stage Stage, total uint, percent float64) error {
	tr.Lock()
	defer tr.Unlock()

	th, err := calculateThreshold(total, percent)
	if err != nil {
		return err
	}

	tr.threshold[stage] = [3]uint{total, th, uint(percent * 100)}

	return nil
}

func (tr *Threshold) MarshalJSON() ([]byte, error) {
	tr.RLock()
	defer tr.RUnlock()

	thh := map[string]interface{}{}
	for k, v := range tr.threshold {
		thh[k.String()] = flattenThreshold(v)
	}

	return json.Marshal(map[string]interface{}{
		"base":      flattenThreshold(tr.base),
		"threshold": thh,
	})
}

func (tr *Threshold) MarshalZerologObject(e *zerolog.Event) {
	tr.RLock()
	defer tr.RUnlock()

	thh := zerolog.Dict()
	for k, v := range tr.threshold {
		thh.Uints(k.String(), v[:])
	}

	e.Uints("base", tr.base[:])
	e.Dict("threshold", thh)
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

func flattenThreshold(a [3]uint) [3]interface{} {
	return [3]interface{}{
		a[0],
		a[1],
		float64(a[2]) / 100,
	}
}
