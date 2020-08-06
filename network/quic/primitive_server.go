package quicnetwork

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lucas-clemente/quic-go/http3"

	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

const QuicEncoderHintHeader string = "x-mitum-encoder-hint"

type PrimitiveQuicServer struct {
	*logging.Logging
	*util.FunctionDaemon
	bind        string
	tlsConfig   *tls.Config
	stoppedChan chan struct{}
	router      *mux.Router
}

func NewPrimitiveQuicServer(bind string, certs []tls.Certificate) (*PrimitiveQuicServer, error) {
	// TODO ratelimit
	qs := &PrimitiveQuicServer{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "network-quic-server")
		}),
		bind: bind,
		tlsConfig: &tls.Config{
			Certificates: certs,
			MinVersion:   tls.VersionTLS13,
			// NextProtos:   []string{""}, // TODO set unique strings
		},
		stoppedChan: make(chan struct{}, 10),
		router:      mux.NewRouter(),
	}

	root := qs.router.Name("root")
	root.Path("/").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		},
	)

	qs.FunctionDaemon = util.NewFunctionDaemon(qs.run, false)

	return qs, nil
}

func (qs *PrimitiveQuicServer) SetHandler(prefix string, handler network.HTTPHandlerFunc) *mux.Route {
	var route *mux.Route
	if prefix == "" || prefix == "/" {
		route = qs.router.Get("root")
	} else {
		route = qs.router.NewRoute()
	}

	return route.Path(prefix).HandlerFunc(handler)
}

func (qs *PrimitiveQuicServer) SetLogger(l logging.Logger) logging.Logger {
	_ = qs.Logging.SetLogger(l)
	_ = qs.FunctionDaemon.SetLogger(l)

	return qs.Log()
}

func (qs *PrimitiveQuicServer) StoppedChan() <-chan struct{} {
	return qs.stoppedChan
}

func (qs *PrimitiveQuicServer) run(stopChan chan struct{}) error {
	qs.Log().Debug().Str("bind", qs.bind).Msg("trying to start server")

	server := &http3.Server{
		Server: &http.Server{
			Addr:      qs.bind,
			TLSConfig: qs.tlsConfig,
			Handler:   network.HTTPLogHandler(qs.router, qs.Log()),
		},
	}

	errChan := make(chan error)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// TODO monkey patch; see https://github.com/lucas-clemente/quic-go/issues/1778
			if err.Error() == "server closed" {
				return
			}

			qs.Log().Error().Err(err).Msg("server failed")

			errChan <- err
		}
	}()

	defer func() {
		qs.stoppedChan <- struct{}{}
	}()

	select {
	case err := <-errChan:
		return err
	case <-stopChan:
		if err := qs.stop(server); err != nil {
			qs.Log().Error().Err(err).Msg("failed to stop server")
			return err
		}
	}

	return nil
}

func (qs *PrimitiveQuicServer) stop(server *http3.Server) error {
	if err := server.Close(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

func EncoderFromHeader(header http.Header, encs *encoder.Encoders, enc encoder.Encoder) (encoder.Encoder, error) {
	s := header.Get(QuicEncoderHintHeader)
	if len(s) < 1 {
		// NOTE if empty header, use default enc
		return enc, nil
	} else if ht, err := hint.NewHintFromString(s); err != nil {
		return nil, err
	} else {
		return encs.Encoder(ht.Type(), ht.Version())
	}
}
