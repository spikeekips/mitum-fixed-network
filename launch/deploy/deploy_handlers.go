package deploy

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/ulule/limiter/v3"
)

var QuicHandlerPathSetBlockDataMaps = "/_deploy/blockdatamaps"

var RateLimitHandlerNameSetBlockDataMaps = "set-blockdatamaps"

type BaseDeployHandler struct {
	*logging.Logging
	handler    func(string) *mux.Route
	handlerMap map[string][]process.RateLimitRule
	store      limiter.Store
	ks         *DeployKeyStorage
	enc        encoder.Encoder
}

func NewBaseDeployHandler(
	ctx context.Context,
	name string,
	handler func(string) *mux.Route,
) (*BaseDeployHandler, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var handlerMap map[string][]process.RateLimitRule
	var store limiter.Store
	if err := process.LoadRateLimitHandlerMapContextValue(ctx, &handlerMap); err != nil {
		handlerMap = map[string][]process.RateLimitRule{}
	} else if err := process.LoadRateLimitStoreContextValue(ctx, &store); err != nil {
		if !errors.Is(err, util.ContextValueNotFoundError) {
			return nil, err
		}
	}

	var ks *DeployKeyStorage
	if err := LoadDeployKeyStorageContextValue(ctx, &ks); err != nil {
		return nil, err
	}

	var enc *jsonenc.Encoder
	if err := config.LoadJSONEncoderContextValue(ctx, &enc); err != nil {
		return nil, err
	}

	dh := &BaseDeployHandler{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", name)
		}),
		handler:    handler,
		handlerMap: handlerMap,
		store:      store,
		ks:         ks,
		enc:        enc,
	}

	_ = dh.SetLogging(log)

	return dh, nil
}

func (dh *BaseDeployHandler) RateLimit(name string, handler http.Handler) http.Handler {
	i, found := dh.handlerMap[name]
	if !found {
		return handler
	}
	return process.NewRateLimitMiddleware(
		process.NewRateLimit(i, limiter.Rate{Limit: -1}),
		dh.store,
	).Middleware(handler)
}

type DeployHandlers struct {
	*BaseDeployHandler
	mw *DeployByKeyMiddleware
}

func NewDeployHandlers(ctx context.Context, handler func(string) *mux.Route) (*DeployHandlers, error) {
	base, err := NewBaseDeployHandler(ctx, "deploy-handlers", handler)
	if err != nil {
		return nil, err
	}

	mw := NewDeployByKeyMiddleware(base.ks)

	dh := &DeployHandlers{
		BaseDeployHandler: base,
		mw:                mw,
	}

	return dh, nil
}

func (dh *DeployHandlers) SetHandler(prefix string, handler http.Handler) *mux.Route {
	return dh.handler(prefix).Handler(dh.mw.Middleware(handler))
}
