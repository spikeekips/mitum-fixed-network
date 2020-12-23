package util

import "sync"

type LockedItem struct {
	sync.RWMutex
	value interface{}
}

func NewLockedItem(defaultValue interface{}) *LockedItem {
	return &LockedItem{value: defaultValue}
}

func (li *LockedItem) Value() interface{} {
	li.RLock()
	defer li.RUnlock()

	return li.value
}

func (li *LockedItem) Set(value interface{}) *LockedItem {
	li.Lock()
	defer li.Unlock()

	li.value = value

	return li
}
