package isaac

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	QuicHandlerPathSendSeal = "/seal"
	QuicHandlerPathGetSeals = "/seals"
)

type QuicServer struct {
	*logging.Logging
	*network.QuicServer
	encs            *encoder.Encoders
	enc             encoder.Encoder // NOTE default encoder.Encoder
	getSealsHandler GetSealsHandler
	newSealHandler  NewSealHandler
}

func NewQuicServer(
	qs *network.QuicServer, encs *encoder.Encoders, enc encoder.Encoder,
) (*QuicServer, error) {
	// TODO ratelimit
	nqs := &QuicServer{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "network-quic-server")
		}),
		QuicServer: qs,
		encs:       encs,
		enc:        enc,
	}
	nqs.setHandlers()

	return nqs, nil
}

func (qs *QuicServer) SetLogger(l logging.Logger) logging.Logger {
	_ = qs.QuicServer.SetLogger(l)

	return qs.Logging.SetLogger(l)
}

func (qs *QuicServer) SetGetSealsHandler(fn GetSealsHandler) {
	qs.getSealsHandler = fn
}

func (qs *QuicServer) SetNewSealHandler(fn NewSealHandler) {
	qs.newSealHandler = fn
}

func (qs *QuicServer) setHandlers() {
	// seal handler
	_ = qs.SetHandler(
		QuicHandlerPathGetSeals,
		func(w http.ResponseWriter, r *http.Request) {
			qs.handleGetSeals(w, r)
		},
	).Methods("POST")

	_ = qs.SetHandler(
		QuicHandlerPathSendSeal,
		func(w http.ResponseWriter, r *http.Request) {
			qs.handleNewSeal(w, r)
		},
	).Methods("POST")

	_ = qs.SetHandler(
		"/",
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)
}

func (qs *QuicServer) handleGetSeals(w http.ResponseWriter, r *http.Request) {
	if qs.getSealsHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		qs.Log().Error().Err(err).Msg("failed to read post body")

		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	var enc encoder.Encoder
	if e, err := network.EncoderFromHeader(r.Header, qs.encs, qs.enc); err != nil {
		qs.Log().Error().Err(err).Msg("failed to read encoder hint")
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else {
		enc = e
	}

	// TODO encoder.Encoder should handle slice
	var hs []json.RawMessage
	if err := enc.Unmarshal(body.Bytes(), &hs); err != nil {
		qs.Log().Error().Err(err).Msg("failed to unmarshal hash slice")
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	var hashes []valuehash.Hash
	for _, r := range hs {
		if hinter, err := enc.DecodeByHint(r); err != nil {
			qs.Log().Error().Err(err).Msg("failed to decode")
			network.HTTPError(w, http.StatusBadRequest)
			return
		} else if h, ok := hinter.(valuehash.Hash); !ok {
			qs.Log().Error().Err(err).Msg("not hash")
			network.HTTPError(w, http.StatusBadRequest)
			return
		} else {
			hashes = append(hashes, h)
		}
	}

	var output []byte
	if sls, err := qs.getSealsHandler(hashes); err != nil {
		qs.Log().Error().Err(err).Msg("failed to get seals")
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else if b, err := qs.enc.Marshal(sls); err != nil {
		qs.Log().Error().Err(err).Msg("failed to encode seals")
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else {
		output = b
	}

	w.Header().Set(network.QuicEncoderHintHeader, qs.enc.Hint().String())
	_, _ = w.Write(output)
}

func (qs *QuicServer) handleNewSeal(w http.ResponseWriter, r *http.Request) {
	if qs.newSealHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		qs.Log().Error().Err(err).Msg("failed to read post body")

		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	var enc encoder.Encoder
	if e, err := network.EncoderFromHeader(r.Header, qs.encs, qs.enc); err != nil {
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else {
		enc = e
	}

	var sl seal.Seal
	if hinter, err := enc.DecodeByHint(body.Bytes()); err != nil {
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else if s, ok := hinter.(seal.Seal); !ok {
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else {
		sl = s
	}

	if err := qs.newSealHandler(sl); err != nil {
		qs.Log().Error().Err(err).Msg("failed to receive new seal")

		network.HTTPError(w, http.StatusInternalServerError)
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
