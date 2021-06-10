package deploy

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/ulule/limiter/v3"
	"golang.org/x/xerrors"
)

var QuicHandlerPathSetBlockDataMaps = "/_deploy/blockdatamaps"

var RateLimitHandlerNameSetBlockDataMaps = "set-blockdatamaps"

type baseDeployHandler struct {
	*logging.Logging
	handler    func(string) *mux.Route
	handlerMap map[string][]process.RateLimitRule
	store      limiter.Store
	ks         *DeployKeyStorage
	enc        encoder.Encoder
}

func newBaseDeployHandler(
	ctx context.Context,
	name string,
	handler func(string) *mux.Route,
) (*baseDeployHandler, error) {
	var log logging.Logger
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return nil, err
	}

	var handlerMap map[string][]process.RateLimitRule
	var store limiter.Store
	if err := process.LoadRateLimitHandlerMapContextValue(ctx, &handlerMap); err != nil {
		handlerMap = map[string][]process.RateLimitRule{}
	} else if err := process.LoadRateLimitStoreContextValue(ctx, &store); err != nil {
		if !xerrors.Is(err, util.ContextValueNotFoundError) {
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

	dh := &baseDeployHandler{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", name)
		}),
		handler:    handler,
		handlerMap: handlerMap,
		store:      store,
		ks:         ks,
		enc:        enc,
	}

	_ = dh.SetLogger(log)

	return dh, nil
}

func (dh *baseDeployHandler) rateLimit(name string, handler http.Handler) http.Handler {
	i, found := dh.handlerMap[name]
	if !found {
		return handler
	}
	return process.NewRateLimitMiddleware(
		process.NewRateLimit(i, limiter.Rate{Limit: -1}),
		dh.store,
	).Middleware(handler)
}

type deployHandlers struct {
	*baseDeployHandler
	db storage.Database
	bc *BlockDataCleaner
	mw *DeployByKeyMiddleware
}

func newDeployHandlers(ctx context.Context, handler func(string) *mux.Route) (*deployHandlers, error) {
	base, err := newBaseDeployHandler(ctx, "deploy-handlers", handler)
	if err != nil {
		return nil, err
	}

	var db storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &db); err != nil {
		return nil, err
	}

	var bc *BlockDataCleaner
	if err := LoadBlockDataCleanerContextValue(ctx, &bc); err != nil {
		return nil, err
	}

	mw := NewDeployByKeyMiddleware(base.ks)

	dh := &deployHandlers{
		baseDeployHandler: base,
		db:                db,
		bc:                bc,
		mw:                mw,
	}

	return dh, nil
}

func (dh *deployHandlers) setHandlers() error {
	setter := []func() error{
		dh.setSetBlockDataMaps,
	}

	for i := range setter {
		if err := setter[i](); err != nil {
			return err
		}
	}

	return nil
}

func (dh *deployHandlers) setSetBlockDataMaps() error {
	handler := dh.rateLimit(
		RateLimitHandlerNameSetBlockDataMaps,
		http.HandlerFunc(NewSetBlockDataMapsHandler(dh.enc, dh.db, dh.bc)),
	)

	_ = dh.handler(QuicHandlerPathSetBlockDataMaps).Handler(handler)

	return nil
}
