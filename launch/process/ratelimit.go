package process

import (
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	limitermemory "github.com/ulule/limiter/v3/drivers/store/memory"
)

type RateLimitRule struct {
	ipnet *net.IPNet
	rate  limiter.Rate
}

func NewRateLimiterRule(ipnet *net.IPNet, rate limiter.Rate) RateLimitRule {
	return RateLimitRule{ipnet: ipnet, rate: rate}
}

func (rr RateLimitRule) Rate() limiter.Rate {
	return rr.rate
}

func (rr RateLimitRule) Match(ip net.IP) bool {
	if rr.ipnet == nil {
		return false
	} else {
		return rr.ipnet.Contains(ip)
	}
}

type RateLimit struct {
	*logging.Logging
	cache       *cache.GCache
	rules       []RateLimitRule
	defaultRate limiter.Rate
}

func NewRateLimit(
	rules []RateLimitRule,
	defaultRate limiter.Rate,
) *RateLimit {
	ca, _ := cache.NewGCache("lru", 100*100, time.Hour*3)

	return &RateLimit{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "ratelimit")
		}),
		cache:       ca,
		rules:       rules,
		defaultRate: defaultRate,
	}
}

func (rl *RateLimit) Rate(ip net.IP) limiter.Rate {
	if i, _ := rl.cache.Get(ip.String()); i != nil {
		return i.(limiter.Rate)
	}

	l := rl.rate(ip)
	_ = rl.cache.Set(ip.String(), l, 0)

	return l
}

func (rl *RateLimit) rate(ip net.IP) limiter.Rate {
	for i := range rl.rules {
		r := rl.rules[i]
		if r.Match(ip) {
			return r.Rate()
		}
	}

	return rl.defaultRate
}

type RateLimitMiddleware struct {
	lt      *RateLimit
	handler http.Handler
	store   limiter.Store
}

func NewRateLimitMiddleware(lt *RateLimit, handler http.Handler, store limiter.Store) *RateLimitMiddleware {
	if store == nil {
		store = limitermemory.NewStoreWithOptions(limiter.StoreOptions{CleanUpInterval: time.Hour})
	}

	return &RateLimitMiddleware{
		lt:      lt,
		handler: handler,
		store:   store,
	}
}

func (mw *RateLimitMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip := limiter.GetIP(r, limiter.Options{TrustForwardHeader: true})
	rate := mw.lt.Rate(ip)
	if rate.Limit < 0 { // NOTE nolimit
		w.Header().Add("X-RateLimit-Limit", "unlimited")

		mw.handler.ServeHTTP(w, r)

		return
	} else if rate.Limit < 1 || rate.Period < 1 {
		network.HTTPError(w, http.StatusTooManyRequests)

		return
	}

	if ctx, err := mw.store.Get(r.Context(), ip.String(), rate); err != nil {
		mw.handler.ServeHTTP(w, r)

		return
	} else {
		w.Header().Add("X-RateLimit-Limit", strconv.FormatInt(ctx.Limit, 10))
		w.Header().Add("X-RateLimit-Remaining", strconv.FormatInt(ctx.Remaining, 10))
		w.Header().Add("X-RateLimit-Reset", strconv.FormatInt(ctx.Reset, 10))

		if ctx.Reached {
			stdlib.DefaultLimitReachedHandler(w, r)

			return
		}
	}

	mw.handler.ServeHTTP(w, r)
}
