package hint

import "sync"

type Hintmap struct {
	sync.RWMutex
	ht *Hintset
	m  *sync.Map
}

func NewHintmap() *Hintmap {
	return &Hintmap{
		ht: NewHintset(),
		m:  &sync.Map{},
	}
}

func (hm *Hintmap) Add(ht Hinter, i interface{}) error {
	if err := hm.ht.Add(ht); err != nil {
		return err
	}

	hm.m.Store(ht.Hint(), i)

	return nil
}

func (hm *Hintmap) Get(ht Hinter) (interface{}, bool) {
	if hinter, err := hm.ht.Hinter(ht.Hint().Type(), ht.Hint().Version()); err != nil {
		return nil, false
	} else {
		return hm.m.Load(hinter.Hint())
	}
}
