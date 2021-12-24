package deploy

import (
	"context"
	"fmt"
	"net/http"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/process"
	"github.com/spikeekips/mitum/network"
	quicnetwork "github.com/spikeekips/mitum/network/quic"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/logging"
)

var HookNameDeployHandlers = "deploy_handlers"

func HookDeployHandlers(ctx context.Context) (context.Context, error) {
	var log *logging.Logging
	if err := config.LoadLogContextValue(ctx, &log); err != nil {
		return ctx, err
	}

	var qnt *quicnetwork.Server
	var nt network.Server
	if err := process.LoadNetworkContextValue(ctx, &nt); err != nil {
		return nil, err
	} else if i, ok := nt.(*quicnetwork.Server); !ok {
		log.Log().Warn().
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

	return hookDefaultDeployHandlers(ctx, qnt)
}

func hookDefaultDeployHandlers(ctx context.Context, qnt *quicnetwork.Server) (context.Context, error) {
	dh, err := NewDeployHandlers(ctx, qnt.Handler)
	if err != nil {
		return ctx, err
	}

	var db storage.Database
	if err := process.LoadDatabaseContextValue(ctx, &db); err != nil {
		return ctx, err
	}

	var bc *BlockdataCleaner
	if err := LoadBlockdataCleanerContextValue(ctx, &bc); err != nil {
		return ctx, err
	}

	setBlockdataMapsHandler := http.HandlerFunc(NewSetBlockdataMapsHandler(qnt.Encoder(), db, bc))
	_ = dh.SetHandler(
		QuicHandlerPathSetBlockdataMaps,
		dh.RateLimit(RateLimitHandlerNameSetBlockdataMaps, setBlockdataMapsHandler),
	)

	return context.WithValue(ctx, ContextValueDeployHandler, dh), nil
}
