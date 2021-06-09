package deploy

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/cache"
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

type deployKeyHandlers struct {
	*baseDeployHandler
	cache cache.Cache
	mw    *DeployKeyByTokenMiddleware
}

func newDeployKeyHandlers(ctx context.Context, handler func(string) *mux.Route) (*deployKeyHandlers, error) {
	var base *baseDeployHandler
	if i, err := newBaseDeployHandler(ctx, "deploy-key-handlers", handler); err != nil {
		return nil, err
	} else {
		base = i
	}

	var c cache.Cache
	if i, err := cache.NewGCache("lru", 100*100, time.Minute); err != nil {
		return nil, xerrors.Errorf("failed to create cache for deploy key handlers")
	} else {
		c = i
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

	dh := &deployKeyHandlers{
		baseDeployHandler: base,
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
	handler := dh.rateLimit(
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
	handler := dh.rateLimit(RateLimitHandlerNameDeployKeyKeys, http.HandlerFunc(NewDeployKeyKeysHandler(dh.ks, dh.enc)))

	_ = dh.handler(QuicHandlerPathDeployKeyKeys).Handler(dh.mw.Middleware(handler))

	return nil
}

func (dh *deployKeyHandlers) setKeyNewHandler() error {
	handler := dh.rateLimit(RateLimitHandlerNameDeployKeyNew, http.HandlerFunc(NewDeployKeyNewHandler(dh.ks, dh.enc)))

	_ = dh.handler(QuicHandlerPathDeployKeyNew).Handler(dh.mw.Middleware(handler))

	return nil
}

func (dh *deployKeyHandlers) keyHandler() network.HTTPHandlerFunc {
	handler := NewDeployKeyKeyHandler(dh.ks, dh.enc)
	return dh.rateLimit(RateLimitHandlerNameDeployKeyKey, http.HandlerFunc(handler)).ServeHTTP
}

func (dh *deployKeyHandlers) keyRevokeHandler() network.HTTPHandlerFunc {
	handler := NewDeployKeyRevokeHandler(dh.ks)
	return dh.rateLimit(RateLimitHandlerNameDeployKeyRevoke, http.HandlerFunc(handler)).ServeHTTP
}