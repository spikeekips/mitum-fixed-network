package network

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/util"
)

const QuicEncoderHintHeader string = "x-mitum-encoder-hint"

type QuicServer struct {
	*logging.Logging
	*util.FunctionDaemon
	bind        string
	tlsConfig   *tls.Config
	stoppedChan chan struct{}
	router      *mux.Router
}

func NewQuicServer(bind string, certs []tls.Certificate) (*QuicServer, error) {
	// TODO ratelimit
	qs := &QuicServer{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "network-quic-server")
		}),
		bind: bind,
		tlsConfig: &tls.Config{
			Certificates: certs,
			// NextProtos:   []string{""}, // TODO set unique strings
		},
		stoppedChan: make(chan struct{}, 10),
		router:      mux.NewRouter(),
	}

	_ = qs.router.HandleFunc(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		},
	)

	qs.FunctionDaemon = util.NewFunctionDaemon(qs.run, false)

	return qs, nil
}

func (qs *QuicServer) SetHandler(prefix string, handler HTTPHandlerFunc) *mux.Route {
	return qs.router.HandleFunc(prefix, handler)
}

func (qs *QuicServer) SetLogger(l logging.Logger) logging.Logger {
	_ = qs.Logging.SetLogger(l)
	_ = qs.FunctionDaemon.SetLogger(l)

	return qs.Log()
}

func (qs *QuicServer) StoppedChan() <-chan struct{} {
	return qs.stoppedChan
}

func (qs *QuicServer) run(stopChan chan struct{}) error {
	qs.Log().Debug().Str("bind", qs.bind).Msg("trying to start server")

	server := &http3.Server{
		Server: &http.Server{
			Addr:      qs.bind,
			TLSConfig: qs.tlsConfig,
			Handler:   HTTPLogHandler(qs.router, qs.Log()),
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

func (qs *QuicServer) stop(server *http3.Server) error {
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
