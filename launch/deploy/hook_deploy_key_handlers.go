package deploy

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/ulule/limiter/v3"
	"golang.org/x/xerrors"
)

var DefaultDeployKeyTokenExpired = time.Second * 30

var (
	QuicHandlerPathDeployKeyKeys      = "/_deploy/key/keys"
	QuicHandlerPathDeployKeyNew       = "/_deploy/key/new"
	QuicHandlerPathDeployKeyToken     = "/_deploy/key/token" // nolint:gosec
	QuicHandlerPathDeployKeyKeyPrefix = "/_deploy/key"
	QuicHandlerPathDeployKeyKey       = QuicHandlerPathDeployKeyKeyPrefix + QuicHandlerPathDeployKeyKeySuffix
)

var (
	RateLimitHandlerNameDeployKeyToken  = "deploy-key-token"
	RateLimitHandlerNameDeployKeyKey    = "deploy-key-key"
	RateLimitHandlerNameDeployKeyKeys   = "deploy-key-keys"
	RateLimitHandlerNameDeployKeyNew    = "deploy-key-new"
	RateLimitHandlerNameDeployKeyRevoke = "deploy-key-revoke"
)

var HookNameDeployKeyHandlers = "deploy_key_handlers"

func HookDeployKeyHandlers(ctx context.Context) (context.Context, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var qnt *quicnetwork.Server
	var nt network.Server
	if err := process.LoadNetworkContextValue(ctx, &nt); err != nil {
		return nil, err
	} else if i, ok := nt.(*quicnetwork.Server); !ok {
		log.Warn().
			Str("network_server_type", fmt.Sprintf("%T", nt)).Msg("only quicnetwork server supports deploy key handlers")

		return ctx, nil
	} else {
		qnt = i
	}

	if i, err := newDeployKeyHandlers(ctx, qnt.Handler); err != nil {
		return ctx, err
	} else if err := i.setHandlers(); err != nil {
		return ctx, err
	}

	return ctx, nil
}

type deployKeyHandlers struct {
	*logging.Logging
	handler    func(string) *mux.Route
	cache      cache.Cache
	handlerMap map[string][]process.RateLimitRule
	store      limiter.Store
	ks         *DeployKeyStorage
	enc        encoder.Encoder
	mw         *DeployKeyByTokenMiddleware
}

func newDeployKeyHandlers(ctx context.Context, handler func(string) *mux.Route) (*deployKeyHandlers, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var c cache.Cache
	if i, err := cache.NewGCache("lru", 100*100, time.Minute); err != nil {
		return nil, xerrors.Errorf("failed to create cache for deploy key handlers")
	} else {
		c = i
	}

	var handlerMap map[string][]process.RateLimitRule
	var store limiter.Store
	if err := process.LoadRateLimitHandlerMapContextValue(ctx, &handlerMap); err != nil {
		handlerMap = map[string][]process.RateLimitRule{}
	} else if err := process.LoadRateLimitStoreContextValue(ctx, &store); err != nil {
		return nil, err
	}

	var ks *DeployKeyStorage
	if err := LoadDeployKeyStorageContextValue(ctx, &ks); err != nil {
		return nil, err
	}

	var enc *jsonenc.Encoder
	if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
		return nil, err
	}

	var local *network.LocalNode
	if err := process.LoadLocalNodeContextValue(ctx, &local); err != nil {
		return nil, err
	}

	var policy *isaac.LocalPolicy
	if err := process.LoadPolicyContextValue(ctx, &policy); err != nil {
		return nil, err
	}

	mw := NewDeployKeyByTokenMiddleware(c, local.Privatekey().Publickey(), policy.NetworkID())

	dk := &deployKeyHandlers{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "deploy-key-handlers")
		}),
		handler:    handler,
		cache:      c,
		handlerMap: handlerMap,
		store:      store,
		ks:         ks,
		enc:        enc,
		mw:         mw,
	}

	_ = dk.SetLogger(log)

	return dk, nil
}

func (dk *deployKeyHandlers) setHandlers() error {
	setter := []func() error{
		dk.setTokenHandler,
		dk.setKeysHandler,
		dk.setKeyNewHandler,
		dk.setKeyHandler,
	}

	for i := range setter {
		if err := setter[i](); err != nil {
			return err
		}
	}

	return nil
}

func (dk *deployKeyHandlers) setTokenHandler() error {
	handler := dk.rateLimit(
		RateLimitHandlerNameDeployKeyToken,
		NewDeployKeyTokenHandler(dk.cache, DefaultDeployKeyTokenExpired),
	)

	_ = dk.handler(QuicHandlerPathDeployKeyToken).Handler(handler)

	return nil
}

func (dk *deployKeyHandlers) setKeyHandler() error {
	getKey := dk.keyHandler()
	revoke := dk.keyRevokeHandler()

	_ = dk.handler(QuicHandlerPathDeployKeyKey).Handler(dk.mw.Middleware(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				getKey(w, r)
			case "DELETE":
				revoke(w, r)
			default:
				network.HTTPError(w, http.StatusMethodNotAllowed)
			}
		},
	)))

	return nil
}

func (dk *deployKeyHandlers) setKeysHandler() error {
	handler := dk.rateLimit(RateLimitHandlerNameDeployKeyKeys, http.HandlerFunc(NewDeployKeyKeysHandler(dk.ks, dk.enc)))

	_ = dk.handler(QuicHandlerPathDeployKeyKeys).Handler(dk.mw.Middleware(handler))

	return nil
}

func (dk *deployKeyHandlers) setKeyNewHandler() error {
	handler := dk.rateLimit(RateLimitHandlerNameDeployKeyNew, http.HandlerFunc(NewDeployKeyNewHandler(dk.ks, dk.enc)))

	_ = dk.handler(QuicHandlerPathDeployKeyNew).Handler(dk.mw.Middleware(handler))

	return nil
}

func (dk *deployKeyHandlers) keyHandler() network.HTTPHandlerFunc {
	handler := NewDeployKeyKeyHandler(dk.ks, dk.enc)
	return dk.rateLimit(RateLimitHandlerNameDeployKeyKey, http.HandlerFunc(handler)).ServeHTTP
}

func (dk *deployKeyHandlers) keyRevokeHandler() network.HTTPHandlerFunc {
	handler := NewDeployKeyRevokeHandler(dk.ks)
	return dk.rateLimit(RateLimitHandlerNameDeployKeyRevoke, http.HandlerFunc(handler)).ServeHTTP
}

func (dk *deployKeyHandlers) rateLimit(name string, handler http.Handler) http.Handler {
	if i, found := dk.handlerMap[name]; !found {
		return handler
	} else {
		return process.NewRateLimitMiddleware(
			process.NewRateLimit(i, limiter.Rate{Limit: -1}),
			dk.store,
		).Middleware(handler)
	}
}
