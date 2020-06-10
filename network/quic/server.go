package quicnetwork

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"path"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	QuicHandlerPathGetSeals     = "/seals"
	QuicHandlerPathSendSeal     = "/seal"
	QuicHandlerPathGetBlocks    = "/blocks"
	QuicHandlerPathGetManifests = "/manifests"
	QuicHandlerPathNodeInfo     = "/"
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
	nodeInfoHandler     network.NodeInfoHandler
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

func (qs *QuicServer) SetGetManifestsHandler(fn network.GetManifestsHandler) {
	qs.getManifestsHandler = fn
}

func (qs *QuicServer) SetGetBlocksHandler(fn network.GetBlocksHandler) {
	qs.getBlocksHandler = fn
}

func (qs *QuicServer) SetNodeInfoHandler(fn network.NodeInfoHandler) {
	qs.nodeInfoHandler = fn
}

func (qs *QuicServer) setHandlers() {
	_ = qs.SetHandler(QuicHandlerPathGetSeals, qs.handleGetSeals).Methods("POST")
	_ = qs.SetHandler(QuicHandlerPathSendSeal, qs.handleNewSeal).Methods("POST")
	_ = qs.SetHandler(QuicHandlerPathGetManifests, qs.handleGetManifests).Methods("POST")
	_ = qs.SetHandler(QuicHandlerPathGetBlocks, qs.handleGetBlocks).Methods("POST")
	_ = qs.SetHandler(QuicHandlerPathNodeInfo, qs.handleNodeInfo)
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

	var args HashesArgs
	if err := enc.Decode(body.Bytes(), &args); err != nil {
		qs.Log().Error().Err(err).Msg("failed to decode")
		network.HTTPError(w, http.StatusBadRequest)
		return
	}

	var output []byte
	if sls, err := qs.getSealsHandler(args.Hashes); err != nil {
		qs.Log().Error().Err(err).Msg("failed to get seals")
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else if b, err := qs.enc.Marshal(sls); err != nil {
		qs.Log().Error().Err(err).Msg("failed to encode seals")
		network.HTTPError(w, http.StatusInternalServerError)
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

	// NOTE if already received, returns 200
	if qs.hasSealHandler != nil {
		if found, err := qs.hasSealHandler(sl.Hash()); err != nil {
			network.HTTPError(w, http.StatusInternalServerError)

			return
		} else if found {
			w.WriteHeader(http.StatusOK)

			return
		}
	}

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

	var args HeightsArgs
	if err := enc.Decode(body.Bytes(), &args); err != nil {
		network.HTTPError(w, http.StatusBadRequest)

		return err
	}

	var output []byte
	if sls, err := getHandler(args.Heights); err != nil {
		network.HTTPError(w, http.StatusBadRequest)

		return err
	} else if b, err := qs.enc.Marshal(sls); err != nil {
		network.HTTPError(w, http.StatusInternalServerError)

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

func (qs *QuicServer) handleNodeInfo(w http.ResponseWriter, _ *http.Request) {
	if qs.nodeInfoHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	var output []byte
	if n, err := qs.nodeInfoHandler(); err != nil {
		qs.Log().Error().Err(err).Msg("failed to get node info")

		network.HTTPError(w, http.StatusInternalServerError)

		return
	} else if b, err := qs.enc.Marshal(n); err != nil {
		qs.Log().Error().Err(err).Msg("failed to encode NodeInfo")

		network.HTTPError(w, http.StatusInternalServerError)

		return
	} else {
		output = b
	}

	w.Header().Set(QuicEncoderHintHeader, qs.enc.Hint().String())
	_, _ = w.Write(output)
}

func mustQuicURL(u, p string) string {
	uu, err := url.Parse(u)
	if err != nil {
		panic(xerrors.Errorf("failed to join quic url: %w", err))
	}

	uu.Path = path.Join(uu.Path, p)

	return uu.String()
}
