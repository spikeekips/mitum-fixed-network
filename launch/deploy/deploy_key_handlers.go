package deploy

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/cache"
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

type deployKeyHandlers struct {
	*BaseDeployHandler
	cache cache.Cache
	mw    *DeployKeyByTokenMiddleware
}

func newDeployKeyHandlers(ctx context.Context, handler func(string) *mux.Route) (*deployKeyHandlers, error) {
	base, err := NewBaseDeployHandler(ctx, "deploy-key-handlers", handler)
	if err != nil {
		return nil, err
	}

	c, err := cache.NewGCache("lru", 100*100, time.Minute)
	if err != nil {
		return nil, errors.Errorf("failed to create cache for deploy key handlers")
	}

	var local node.Local
	if err := process.LoadLocalNodeContextValue(ctx, &local); err != nil {
		return nil, err
	}

	var policy *isaac.LocalPolicy
	if err := process.LoadPolicyContextValue(ctx, &policy); err != nil {
		return nil, err
	}

	mw := NewDeployKeyByTokenMiddleware(c, local.Privatekey().Publickey(), policy.NetworkID())

	dh := &deployKeyHandlers{
		BaseDeployHandler: base,
		cache:             c,
		mw:                mw,
	}

	return dh, nil
}

func (dh *deployKeyHandlers) setHandlers() error {
	setter := []func() error{
		dh.setTokenHandler,
		dh.setKeysHandler,
		dh.setKeyNewHandler,
		dh.setKeyHandler,
	}

	for i := range setter {
		if err := setter[i](); err != nil {
			return err
		}
	}

	return nil
}

func (dh *deployKeyHandlers) setTokenHandler() error {
	handler := dh.RateLimit(
		RateLimitHandlerNameDeployKeyToken,
		NewDeployKeyTokenHandler(dh.cache, DefaultDeployKeyTokenExpired),
	)

	_ = dh.handler(QuicHandlerPathDeployKeyToken).Handler(handler)

	return nil
}

func (dh *deployKeyHandlers) setKeyHandler() error {
	getKey := dh.keyHandler()
	revoke := dh.keyRevokeHandler()

	_ = dh.handler(QuicHandlerPathDeployKeyKey).Handler(dh.mw.Middleware(http.HandlerFunc(
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

func (dh *deployKeyHandlers) setKeysHandler() error {
	handler := dh.RateLimit(RateLimitHandlerNameDeployKeyKeys, http.HandlerFunc(NewDeployKeyKeysHandler(dh.ks, dh.enc)))

	_ = dh.handler(QuicHandlerPathDeployKeyKeys).Handler(dh.mw.Middleware(handler))

	return nil
}

func (dh *deployKeyHandlers) setKeyNewHandler() error {
	handler := dh.RateLimit(RateLimitHandlerNameDeployKeyNew, http.HandlerFunc(NewDeployKeyNewHandler(dh.ks, dh.enc)))

	_ = dh.handler(QuicHandlerPathDeployKeyNew).Handler(dh.mw.Middleware(handler))

	return nil
}

func (dh *deployKeyHandlers) keyHandler() network.HTTPHandlerFunc {
	handler := NewDeployKeyKeyHandler(dh.ks, dh.enc)
	return dh.RateLimit(RateLimitHandlerNameDeployKeyKey, http.HandlerFunc(handler)).ServeHTTP
}

func (dh *deployKeyHandlers) keyRevokeHandler() network.HTTPHandlerFunc {
	handler := NewDeployKeyRevokeHandler(dh.ks)
	return dh.RateLimit(RateLimitHandlerNameDeployKeyRevoke, http.HandlerFunc(handler)).ServeHTTP
}
