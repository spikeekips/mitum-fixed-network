package cache

import (
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/bluele/gcache"
	"github.com/pkg/errors"
)

var (
	DefaultGCacheSize = 100 * 100
	DefaultGCacheType = "lru"
)

type GCache struct {
	gc     gcache.Cache
	tp     string
	size   int
	expire time.Duration
}

func NewGCacheWithQuery(config url.Values) (*GCache, error) {
	size := DefaultGCacheSize
	expire := DefaultCacheExpire
	tp := DefaultGCacheType

	if config != nil && len(config) > 0 {
		if s := config.Get("size"); len(s) > 0 {
			if n, err := strconv.ParseInt(s, 10, 32); err != nil {
				return nil, errors.Wrapf(err, "invalid size, %q of GCache", s)
			} else if n > 0 && n <= math.MaxInt32 {
				size = int(n)
			}
		}

		if s := config.Get("expire"); len(s) > 0 {
			n, err := time.ParseDuration(s)
			if err != nil {
				return nil, errors.Wrapf(err, "invalid expire, %qof GCache", s)
			}
			expire = n
		}

		if s := config.Get("type"); len(s) > 0 {
			switch s {
			case "lru", "lfu", "arc":
				tp = s
			default:
				return nil, errors.Errorf("not supported type, %q of GCache", s)
			}
		}
	}

	return NewGCache(tp, size, expire)
}

func NewGCache(tp string, size int, expire time.Duration) (*GCache, error) {
	builder := gcache.New(size)
	switch tp {
	case "lru":
		builder = builder.LRU()
	case "lfu":
		builder = builder.LFU()
	case "arc":
		builder = builder.ARC()
	default:
		return nil, errors.Errorf("not supported type, %q of GCache", tp)
	}

	gc := builder.Expiration(expire).Build()

	return &GCache{
		gc:     gc,
		tp:     tp,
		size:   size,
		expire: expire,
	}, nil
}

func (ca *GCache) Has(key interface{}) bool {
	return ca.gc.Has(key)
}

func (ca *GCache) Get(key interface{}) (interface{}, error) {
	return ca.gc.Get(key)
}

func (ca *GCache) Set(key interface{}, b interface{}, expire time.Duration) error {
	if expire <= 0 {
		expire = ca.expire
	}

	return ca.gc.SetWithExpire(key, b, expire)
}

func (ca *GCache) SetWithoutExpire(key interface{}, b interface{}) error {
	return ca.gc.Set(key, b)
}

func (ca *GCache) Remove(key interface{}) bool {
	return ca.gc.Remove(key)
}

func (ca *GCache) Purge() error {
	ca.gc.Purge()

	return nil
}

func (ca *GCache) New() (Cache, error) {
	return NewGCache(ca.tp, ca.size, ca.expire)
}

func (ca *GCache) Traverse(callback func(k, v interface{}) bool) error {
	keys := ca.gc.Keys(true)

	for i := range keys {
		k := keys[i]
		j, err := ca.gc.Get(k)
		if err != nil {
			if errors.Is(err, gcache.KeyNotFoundError) {
				continue
			}

			return err
		}

		if !callback(k, j) {
			break
		}
	}

	return nil
}
