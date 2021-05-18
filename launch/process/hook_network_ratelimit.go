package process

import (
	"context"
	"fmt"

	"github.com/spikeekips/mitum/launch/config"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/ulule/limiter/v3"
)

const HookNameNetworkRateLimit = "network_ratelimit"

func HookNetworkRateLimit(ctx context.Context) (context.Context, error) {
	var localconf config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &localconf); err != nil {
		return nil, err
	}

	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	conf := localconf.Network().RateLimit()
	if conf == nil {
		log.Debug().Msg("ratelimit disabled")

		return ctx, nil
	}

	var store limiter.Store
	if conf.Cache() != nil {
		if i, err := quicnetwork.RateLimitStoreFromURI(conf.Cache().String()); err != nil {
			return ctx, err
		} else {
			log.Debug().Str("store", conf.Cache().String()).Msg("ratelimit store created")

			store = i
		}
	}

	var nt *quicnetwork.Server
	if err := util.LoadFromContextValue(ctx, ContextValueNetwork, &nt); err != nil {
		return ctx, err
	}

	rules := conf.Rules()

	handlerMap := map[string][]RateLimitRule{}
	for i := range rules {
		r := rules[i]

		rs := r.Rules()
		for j := range rs {
			log.Debug().
				Str("handler", j).
				Str("target", r.Target()).
				Str("limit", fmt.Sprintf("%d/%s", rs[j].Limit, rs[j].Period.String())).
				Msg("found ratelimit of handler")

			handlerMap[j] = append(handlerMap[j], NewRateLimiterRule(r.IPNet(), rs[j]))
		}
	}

	for i := range handlerMap {
		if err := attachRateLimitToHandler(ctx, i, handlerMap[i], nt, store); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

func attachRateLimitToHandler(
	ctx context.Context,
	name string,
	rules []RateLimitRule,
	nt *quicnetwork.Server,
	store limiter.Store,
) error {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return err
	}

	l := log.WithLogger(func(lctx logging.Context) logging.Emitter {
		return lctx.Str("handler", name)
	})

	var prefix string
	if len(rules) < 1 {
		l.Warn().Msg("empty rule; ignored")

		return nil
	} else if j, found := config.RateLimitHandlerMap[name]; !found {
		l.Warn().Msg("unknown handler found; ignored")

		return nil
	} else {
		prefix = j
	}

	handler := nt.Handler(prefix)

	mw := NewRateLimitMiddleware(
		NewRateLimit(rules, limiter.Rate{Limit: -1}), // NOTE by default, unlimited
		handler.GetHandler(),
		store,
	)

	_ = nt.SetHandler(prefix, mw)

	log.Debug().Str("prefix", prefix).Msg("ratelimit middleware attached")

	return nil
}
