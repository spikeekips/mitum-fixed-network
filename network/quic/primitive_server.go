package quicnetwork

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

const QuicEncoderHintHeader string = "X-MITUM-ENCODER-HINT"

type PrimitiveQuicServer struct {
	*logging.Logging
	*util.ContextDaemon
	bind        string
	tlsConfig   *tls.Config
	stoppedChan chan struct{}
	router      *mux.Router
	httpLog     *logging.Logging
}

func NewPrimitiveQuicServer(
	bind string,
	certs []tls.Certificate,
	httpLog *logging.Logging,
) (*PrimitiveQuicServer, error) {
	if err := network.CheckBindIsOpen("udp", bind, time.Second*1); err != nil {
		return nil, errors.Wrapf(err, "failed to open quic server, %q", bind)
	}

	qs := &PrimitiveQuicServer{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "network-quic-primitive-server")
		}),
		bind: bind,
		tlsConfig: &tls.Config{
			Certificates: certs,
			MinVersion:   tls.VersionTLS13,
		},
		stoppedChan: make(chan struct{}, 10),
		router:      mux.NewRouter(),
		httpLog:     httpLog,
	}

	root := qs.router.Name("root")
	root.Path("/").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		},
	)

	qs.ContextDaemon = util.NewContextDaemon("network-quic-primitive-server", qs.run)

	return qs, nil
}

func (qs *PrimitiveQuicServer) Handler(prefix string) *mux.Route {
	var route *mux.Route
	if prefix == "" || prefix == "/" {
		route = qs.router.Get("root")
	} else if i := qs.router.Get(prefix); i == nil {
		route = qs.router.Name(prefix).Path(prefix)
	} else {
		route = i
	}

	return route
}

func (qs *PrimitiveQuicServer) SetHandlerFunc(prefix string, f network.HTTPHandlerFunc) *mux.Route {
	return qs.SetHandler(prefix, http.HandlerFunc(f))
}

func (qs *PrimitiveQuicServer) SetHandler(prefix string, handler http.Handler) *mux.Route {
	return qs.Handler(prefix).Handler(handler)
}

func (qs *PrimitiveQuicServer) SetLogging(l *logging.Logging) *logging.Logging {
	_ = qs.ContextDaemon.SetLogging(l)

	return qs.Logging.SetLogging(l)
}

func (qs *PrimitiveQuicServer) StoppedChan() <-chan struct{} {
	return qs.stoppedChan
}

func (qs *PrimitiveQuicServer) run(ctx context.Context) error {
	qs.Log().Debug().Str("bind", qs.bind).Msg("trying to start server")

	var log *zerolog.Logger
	if qs.httpLog != nil {
		log = qs.httpLog.Log()
	} else {
		log = qs.Log()
	}

	server := &http3.Server{
		Server: &http.Server{
			Addr:      qs.bind,
			TLSConfig: qs.tlsConfig,
			Handler:   network.HTTPLogHandler(qs.router, log),
		},
	}

	errChan := make(chan error)
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			// NOTE monkey patch; see https://github.com/lucas-clemente/quic-go/issues/1778
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
	case <-ctx.Done():
		if err := qs.stop(server); err != nil { // nolint:contextcheck
			qs.Log().Error().Err(err).Msg("failed to stop server")
			return err
		}
	}

	return nil
}

func (*PrimitiveQuicServer) stop(server *http3.Server) error {
	if err := server.Close(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	return server.Shutdown(ctx)
}

func EncoderFromHeader(header http.Header, encs *encoder.Encoders, enc encoder.Encoder) (encoder.Encoder, error) {
	s := header.Get(QuicEncoderHintHeader)
	if len(s) < 1 {
		// NOTE if empty header, use default enc
		return enc, nil
	} else if ht, err := hint.ParseHint(s); err != nil {
		return nil, err
	} else {
		return encs.Encoder(ht.Type(), ht.Version())
	}
}
