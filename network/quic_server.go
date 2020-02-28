package network

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/gorilla/mux"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	QuicHandlerPathSendSeal = "/seal"
	QuicHandlerPathGetSeals = "/seals"
)

type QuicServer struct {
	*logging.Logger
	*util.FunctionDaemon
	encs           *encoder.Encoders
	enc            encoder.Encoder // NOTE default encoder.Encoder
	bind           string
	tlsConfig      *tls.Config
	stoppedChan    chan struct{}
	router         *mux.Router
	getSealHandler GetSealsHandler
	newSealHandler NewSealHandler
}

func NewQuicServer(
	bind string, certs []tls.Certificate, encs *encoder.Encoders, enc encoder.Encoder,
) (*QuicServer, error) {
	// TODO ratelimit
	qs := &QuicServer{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "network-quic-server")
		}),
		bind: bind,
		encs: encs,
		enc:  enc,
		tlsConfig: &tls.Config{
			Certificates: certs,
			// NextProtos:   []string{""}, // TODO set NetworkID
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

func (qs *QuicServer) SetGetSealHandler(fn GetSealsHandler) {
	qs.getSealHandler = fn
}

func (qs *QuicServer) SetNewSealHandler(fn NewSealHandler) {
	qs.newSealHandler = fn
}

func (qs *QuicServer) setHandler(prefix string, handler HTTPHandlerFunc) *mux.Route {
	return qs.router.HandleFunc(prefix, handler)
}

func (qs *QuicServer) SetLogger(l zerolog.Logger) *logging.Logger {
	_ = qs.Logger.SetLogger(l)
	_ = qs.FunctionDaemon.SetLogger(l)

	return qs.Logger
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
	if err := qs.setHandlers(); err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			// TODO monkey patch
			if err.Error() == "server closed" {
				return
			}

			qs.Log().Error().Err(err).Msg("failed to start server")
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

func (qs *QuicServer) setHandlers() error {
	// seal handler
	_ = qs.setHandler(
		QuicHandlerPathGetSeals,
		func(w http.ResponseWriter, r *http.Request) {
			qs.handleGetSeals(w, r)
		},
	).Methods("POST")

	_ = qs.setHandler(
		QuicHandlerPathSendSeal,
		func(w http.ResponseWriter, r *http.Request) {
			qs.handleNewSeal(w, r)
		},
	).Methods("POST")

	_ = qs.setHandler(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)

	return nil
}

func (qs *QuicServer) handleGetSeals(w http.ResponseWriter, r *http.Request) {
	if qs.getSealHandler == nil {
		HTTPError(w, http.StatusInternalServerError)
		return
	}

	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		qs.Log().Error().Err(err).Msg("failed to read post body")

		HTTPError(w, http.StatusInternalServerError)
		return
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(r.Header, qs.encs, qs.enc); err != nil {
		qs.Log().Error().Err(err).Msg("failed to read encoder hint")
		HTTPError(w, http.StatusBadRequest)
		return
	} else {
		enc = e
	}

	// TODO encoder.Encoder should handle slice
	var hs []json.RawMessage
	if err := enc.Unmarshal(body.Bytes(), &hs); err != nil {
		qs.Log().Error().Err(err).Msg("failed to unmarshal hash slice")
		HTTPError(w, http.StatusInternalServerError)
		return
	}

	var hashes []valuehash.Hash
	for _, r := range hs {
		if hinter, err := enc.DecodeByHint(r); err != nil {
			qs.Log().Error().Err(err).Msg("failed to decode")
			HTTPError(w, http.StatusBadRequest)
			return
		} else if h, ok := hinter.(valuehash.Hash); !ok {
			qs.Log().Error().Err(err).Msg("not hash")
			HTTPError(w, http.StatusBadRequest)
			return
		} else {
			hashes = append(hashes, h)
		}
	}

	var output []byte
	if sls, err := qs.getSealHandler(hashes); err != nil {
		qs.Log().Error().Err(err).Msg("failed to get seals")
		HTTPError(w, http.StatusBadRequest)
		return
	} else if b, err := qs.enc.Marshal(sls); err != nil {
		qs.Log().Error().Err(err).Msg("failed to encode seals")
		HTTPError(w, http.StatusBadRequest)
		return
	} else {
		output = b
	}

	w.Header().Set(QuicEncoderHintHeader, qs.enc.Hint().String())
	_, _ = w.Write(output)
}

func (qs *QuicServer) handleNewSeal(w http.ResponseWriter, r *http.Request) {
	if qs.newSealHandler == nil {
		HTTPError(w, http.StatusInternalServerError)
		return
	}

	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		qs.Log().Error().Err(err).Msg("failed to read post body")

		HTTPError(w, http.StatusInternalServerError)
		return
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(r.Header, qs.encs, qs.enc); err != nil {
		HTTPError(w, http.StatusBadRequest)
		return
	} else {
		enc = e
	}

	var sl seal.Seal
	if hinter, err := enc.DecodeByHint(body.Bytes()); err != nil {
		HTTPError(w, http.StatusBadRequest)
		return
	} else if s, ok := hinter.(seal.Seal); !ok {
		HTTPError(w, http.StatusBadRequest)
		return
	} else {
		sl = s
	}

	if err := qs.newSealHandler(sl); err != nil {
		qs.Log().Error().Err(err).Msg("failed to receive new seal")

		HTTPError(w, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func QuicGetSealsURL(u string) (string, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	uu.Path = path.Join(uu.Path, QuicHandlerPathGetSeals)

	return uu.String(), nil
}

func QuicSendSealURL(u string) (string, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	uu.Path = path.Join(uu.Path, QuicHandlerPathSendSeal)

	return uu.String(), nil
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
