package hint

import (
	"sync"
)

type Hintmap struct {
	sync.RWMutex
	hs *Hintset
	m  map[string]interface{}
}

func NewHintmap() *Hintmap {
	return &Hintmap{
		hs: NewHintset(),
		m:  map[string]interface{}{},
	}
}

func (hm *Hintmap) Add(ht Hinter, i interface{}) error {
	hm.Lock()
	defer hm.Unlock()

	if err := hm.hs.Add(ht); err != nil {
		return err
	}

	hm.m[ht.Hint().String()] = i

	return nil
}

func (hm *Hintmap) Compatible(ht Hinter) (interface{}, error) {
	hm.RLock()
	defer hm.RUnlock()

	return hm.compatible(ht.Hint())
}

func (hm *Hintmap) CompatibleByHint(ht Hint) (interface{}, error) {
	hm.RLock()
	defer hm.RUnlock()

	return hm.compatible(ht)
}

func (hm *Hintmap) compatible(ht Hint) (interface{}, error) {
	hinter, err := hm.hs.Compatible(ht)
	if err != nil {
		return nil, err
	}
	return hm.m[hinter.Hint().String()], nil
}
