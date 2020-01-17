package encoder

import (
	"reflect"
	"sync"
)

type cache struct {
	c *sync.Map
}

func newCache() *cache {
	return &cache{c: &sync.Map{}}
}

func (ec *cache) Get(key interface{}) (interface{}, bool) {
	return ec.c.Load(key)
}

func (ec *cache) Set(key, value interface{}) {
	ec.c.Store(key, value)
}

func (ec *cache) Delete(key interface{}) {
	ec.c.Delete(key)
}

type CachedPacker struct {
	Type   reflect.Type
	Pack   interface{}
	Unpack interface{}
}

func NewCachedPacker(t reflect.Type, pack, unpack interface{}) CachedPacker {
	return CachedPacker{Type: t, Pack: pack, Unpack: unpack}
}
