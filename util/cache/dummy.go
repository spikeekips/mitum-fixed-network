package cache

import (
	"time"
)

type Dummy struct {
}

func (ca Dummy) Has(interface{}) bool {
	return false
}

func (ca Dummy) Get(interface{}) (interface{}, error) {
	return nil, nil
}

func (ca Dummy) Set(interface{}, interface{}, time.Duration) error {
	return nil
}

func (ca Dummy) Purge() error {
	return nil
}

func (ca Dummy) New() (Cache, error) {
	return Dummy{}, nil
}
