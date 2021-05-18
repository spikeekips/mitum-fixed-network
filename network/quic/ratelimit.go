package quicnetwork

import (
	"net/url"
	"time"

	libredis "github.com/go-redis/redis/v8"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"github.com/ulule/limiter/v3/drivers/store/redis"
	"golang.org/x/xerrors"
)

func RateLimitStoreFromURI(s string) (limiter.Store, error) {
	var u *url.URL
	if i, err := url.Parse(s); err != nil {
		return nil, xerrors.Errorf("wrong ratelimit cache url, %q", s)
	} else {
		u = i
	}

	var prefix string = "mitum:limiter"
	if i := u.Query().Get("prefix"); len(i) > 0 {
		prefix = i
	}

	switch {
	case u.Scheme == "memory":
		if i, err := newMemoryRateLimitStore(u, prefix); err != nil {
			return nil, xerrors.Errorf("failed to create ratelimit memory store: %w", err)
		} else {
			return i, nil
		}
	case u.Scheme == "redis":
		if i, err := newRedisRateLimitStore(u, prefix); err != nil {
			return nil, xerrors.Errorf("failed to create ratelimit redis store: %w", err)
		} else {
			return i, nil
		}
	default:
		return nil, xerrors.Errorf("unknown ratelimit cache uri: %q", u.String())
	}
}

func newMemoryRateLimitStore(u *url.URL, prefix string) (limiter.Store, error) {
	var cleanup time.Duration = limiter.DefaultCleanUpInterval
	if i := u.Query().Get("cleanup-interval"); len(i) > 0 {
		if d, err := time.ParseDuration(i); err != nil {
			return nil, err
		} else {
			cleanup = d
		}
	}

	return memory.NewStoreWithOptions(limiter.StoreOptions{
		Prefix:          prefix,
		CleanUpInterval: cleanup,
	}), nil
}

func newRedisRateLimitStore(u *url.URL, prefix string) (limiter.Store, error) {
	var client *libredis.Client
	if i, err := libredis.ParseURL(u.String()); err != nil {
		return nil, err
	} else {
		client = libredis.NewClient(i)
	}

	return redis.NewStoreWithOptions(client, limiter.StoreOptions{
		Prefix: prefix,
	})
}
