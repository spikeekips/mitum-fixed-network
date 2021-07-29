package cache

import (
	"time"
)

type Dummy struct{}

func (Dummy) Has(interface{}) bool {
	return false
}

func (Dummy) Get(interface{}) (interface{}, error) {
	return nil, nil
}

func (Dummy) Set(interface{}, interface{}, time.Duration) error {
	return nil
}

func (Dummy) Purge() error {
	return nil
}

func (Dummy) Remove(interface{}) bool {
	return false
}

func (Dummy) New() (Cache, error) {
	return Dummy{}, nil
}
