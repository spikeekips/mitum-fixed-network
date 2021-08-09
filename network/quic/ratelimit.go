package quicnetwork

import (
	"net/url"
	"time"

	libredis "github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/network"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"github.com/ulule/limiter/v3/drivers/store/redis"
)

func RateLimitStoreFromURI(s string) (limiter.Store, error) {
	u, err := network.ParseURL(s, false)
	if err != nil {
		return nil, errors.Errorf("wrong ratelimit cache url, %q", s)
	}

	prefix := "mitum:limiter"
	if i := u.Query().Get("prefix"); len(i) > 0 {
		prefix = i
	}

	switch {
	case u.Scheme == "memory":
		i, err := newMemoryRateLimitStore(u, prefix)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create ratelimit memory store")
		}
		return i, nil
	case u.Scheme == "redis":
		i, err := newRedisRateLimitStore(u, prefix)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create ratelimit redis store")
		}
		return i, nil
	default:
		return nil, errors.Errorf("unknown ratelimit cache uri: %q", u.String())
	}
}

func newMemoryRateLimitStore(u *url.URL, prefix string) (limiter.Store, error) {
	cleanup := limiter.DefaultCleanUpInterval
	if i := u.Query().Get("cleanup-interval"); len(i) > 0 {
		d, err := time.ParseDuration(i)
		if err != nil {
			return nil, err
		}
		cleanup = d
	}

	return memory.NewStoreWithOptions(limiter.StoreOptions{
		Prefix:          prefix,
		CleanUpInterval: cleanup,
	}), nil
}

func newRedisRateLimitStore(u *url.URL, prefix string) (limiter.Store, error) {
	u.RawQuery = ""
	i, err := libredis.ParseURL(u.String())
	if err != nil {
		return nil, err
	}
	client := libredis.NewClient(i)

	return redis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix: prefix,
	})
}
