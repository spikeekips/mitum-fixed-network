package quicnetwork

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	QuicHandlerPathSendSeal     = "/seal"
	QuicHandlerPathGetSeals     = "/seals"
	QuicHandlerPathGetBlocks    = "/blocks"
	QuicHandlerPathGetManifests = "/manifests"
)

type QuicServer struct {
	*logging.Logging
	*PrimitiveQuicServer
	encs                *encoder.Encoders
	enc                 encoder.Encoder // NOTE default encoder.Encoder
	getSealsHandler     network.GetSealsHandler
	hasSealHandler      network.HasSealHandler
	newSealHandler      network.NewSealHandler
	getManifestsHandler network.GetManifestsHandler
	getBlocksHandler    network.GetBlocksHandler
}

func NewQuicServer(
	prim *PrimitiveQuicServer, encs *encoder.Encoders, enc encoder.Encoder,
) (*QuicServer, error) {
	// TODO ratelimit
	nqs := &QuicServer{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "network-quic-server")
		}),
		PrimitiveQuicServer: prim,
		encs:                encs,
		enc:                 enc,
	}
	nqs.setHandlers()

	return nqs, nil
}

func (qs *QuicServer) SetLogger(l logging.Logger) logging.Logger {
	_ = qs.PrimitiveQuicServer.SetLogger(l)

	return qs.Logging.SetLogger(l)
}

func (qs *QuicServer) SetHasSealHandler(fn network.HasSealHandler) {
	qs.hasSealHandler = fn
}

func (qs *QuicServer) SetGetSealsHandler(fn network.GetSealsHandler) {
	qs.getSealsHandler = fn
}

func (qs *QuicServer) SetNewSealHandler(fn network.NewSealHandler) {
	qs.newSealHandler = fn
}

func (qs *QuicServer) SetGetManifests(fn network.GetManifestsHandler) {
	qs.getManifestsHandler = fn
}

func (qs *QuicServer) SetGetBlocks(fn network.GetBlocksHandler) {
	qs.getBlocksHandler = fn
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
		QuicHandlerPathGetManifests,
		func(w http.ResponseWriter, r *http.Request) {
			qs.handleGetManifests(w, r)
		},
	).Methods("POST")

	_ = qs.SetHandler(
		QuicHandlerPathGetBlocks,
		func(w http.ResponseWriter, r *http.Request) {
			qs.handleGetBlocks(w, r)
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
	if e, err := EncoderFromHeader(r.Header, qs.encs, qs.enc); err != nil {
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

	w.Header().Set(QuicEncoderHintHeader, qs.enc.Hint().String())
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
	if e, err := EncoderFromHeader(r.Header, qs.encs, qs.enc); err != nil {
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else {
		enc = e
	}

	var sl seal.Seal
	if s, err := seal.DecodeSeal(enc, body.Bytes()); err != nil {
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else {
		sl = s
	}

	// NOTE if already received seal, returns 200
	if qs.hasSealHandler != nil {
		if found, err := qs.hasSealHandler(sl.Hash()); err != nil {
			network.HTTPError(w, http.StatusInternalServerError)

			return
		} else if found {
			w.WriteHeader(http.StatusOK)

			return
		}
	}

	// TODO If node is not in consensus state, node will return
	// 425(StatusTooEarly) for new incoming seal.
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/425

	if err := qs.newSealHandler(sl); err != nil {
		seal.LoggerWithSeal(
			sl,
			qs.Log().Error().Err(err),
			qs.Log().IsVerbose(),
		).Msg("failed to receive new seal")

		network.HTTPError(w, http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (qs *QuicServer) handleGetByHeights(
	w http.ResponseWriter,
	r *http.Request,
	getHandler func([]base.Height) (interface{}, error),
) error {
	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		qs.Log().Error().Err(err).Msg("failed to read post body")
		network.HTTPError(w, http.StatusInternalServerError)

		return err
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(r.Header, qs.encs, qs.enc); err != nil {
		network.HTTPError(w, http.StatusBadRequest)

		return xerrors.Errorf("failed to read encoder hint: %w", err)
	} else {
		enc = e
	}

	var heights []base.Height
	if err := enc.Unmarshal(body.Bytes(), &heights); err != nil {
		network.HTTPError(w, http.StatusInternalServerError)

		return xerrors.Errorf("failed to unmarshal hash slice: %w", err)
	}

	var output []byte
	if sls, err := getHandler(heights); err != nil {
		network.HTTPError(w, http.StatusBadRequest)

		return err
	} else if b, err := qs.enc.Marshal(sls); err != nil {
		network.HTTPError(w, http.StatusBadRequest)

		return xerrors.Errorf("failed to encode: %w", err)
	} else {
		output = b
	}

	w.Header().Set(QuicEncoderHintHeader, qs.enc.Hint().String())
	_, _ = w.Write(output)

	return nil
}

func (qs *QuicServer) handleGetManifests(w http.ResponseWriter, r *http.Request) {
	if qs.getManifestsHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	if err := qs.handleGetByHeights(
		w, r,
		func(heights []base.Height) (interface{}, error) {
			return qs.getManifestsHandler(heights)
		},
	); err != nil {
		qs.Log().Error().Err(err).Msg("failed to get manifests")
		return
	}
}

func (qs *QuicServer) handleGetBlocks(w http.ResponseWriter, r *http.Request) {
	if qs.getBlocksHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	if err := qs.handleGetByHeights(
		w, r,
		func(heights []base.Height) (interface{}, error) {
			return qs.getBlocksHandler(heights)
		}); err != nil {
		qs.Log().Error().Err(err).Msg("failed to get blocks")
		return
	}
}

func quicURL(u, p string) (string, error) {
	uu, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	uu.Path = path.Join(uu.Path, p)

	return uu.String(), nil
}
