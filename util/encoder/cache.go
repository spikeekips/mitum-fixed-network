package encoder

import (
	"reflect"
	"sync"
)

type Cache struct {
	c *sync.Map
}

func NewCache() *Cache {
	return &Cache{c: &sync.Map{}}
}

func (ec *Cache) Get(key interface{}) (interface{}, bool) {
	return ec.c.Load(key)
}

func (ec *Cache) Set(key, value interface{}) {
	ec.c.Store(key, value)
}

func (ec *Cache) Delete(key interface{}) {
	ec.c.Delete(key)
}

type CachedPacker struct {
	Type   reflect.Type
	Unpack interface{}
}

func NewCachedPacker(t reflect.Type, unpack interface{}) CachedPacker {
	return CachedPacker{Type: t, Unpack: unpack}
}
