package cache

import (
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/bluele/gcache"
	"golang.org/x/xerrors"
)

var (
	DefaultGCacheSize int = 100 * 100
	DefaultGCacheType     = "lru"
)

type GCache struct {
	gc     gcache.Cache
	tp     string
	size   int
	expire time.Duration
}

func NewGCacheWithQuery(config url.Values) (*GCache, error) {
	var size int = DefaultGCacheSize
	var expire time.Duration = DefaultCacheExpire
	var tp string = DefaultGCacheType

	if config != nil && len(config) > 0 {
		if s := config.Get("size"); len(s) > 0 {
			if n, err := strconv.ParseInt(s, 10, 32); err != nil {
				return nil, xerrors.Errorf("invalid size, %q of GCache: %w", s, err)
			} else if n > 0 && n <= math.MaxInt32 {
				size = int(n)
			}
		}

		if s := config.Get("expire"); len(s) > 0 {
			if n, err := time.ParseDuration(s); err != nil {
				return nil, xerrors.Errorf("invalid expire, %qof GCache: %w", s, err)
			} else {
				expire = n
			}
		}

		if s := config.Get("type"); len(s) > 0 {
			switch s {
			case "lru", "lfu", "arc":
				tp = s
			default:
				return nil, xerrors.Errorf("unsupported type, %q of GCache", s)
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
		return nil, xerrors.Errorf("unsupported type, %q of GCache", tp)
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

func (ca *GCache) Purge() error {
	ca.gc.Purge()

	return nil
}

func (ca *GCache) New() (Cache, error) {
	return NewGCache(ca.tp, ca.size, ca.expire)
}
