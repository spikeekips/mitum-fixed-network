package quicnetwork

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	DefaultPort                 = "54321"
	QuicHandlerPathGetSeals     = "/seals"
	QuicHandlerPathSendSeal     = "/seal"
	QuicHandlerPathGetBlocks    = "/blocks"
	QuicHandlerPathGetManifests = "/manifests"
	QuicHandlerPathNodeInfo     = "/"
)

var cacheKeyNodeInfo = [2]byte{0x00, 0x00}

type Server struct {
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

	cache *cache.GCache
}

func NewServer(
	prim *PrimitiveQuicServer, encs *encoder.Encoders, enc encoder.Encoder,
) (*Server, error) {
	var ca *cache.GCache
	if c, err := cache.NewGCache("lru", 100, time.Second*3); err != nil {
		return nil, err
	} else {
		ca = c
	}

	nqs := &Server{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "network-quic-server")
		}),
		PrimitiveQuicServer: prim,
		encs:                encs,
		enc:                 enc,
		cache:               ca,
	}
	nqs.setHandlers()

	return nqs, nil
}

func (sv *Server) Initialize() error {
	return nil
}

func (sv *Server) SetLogger(l logging.Logger) logging.Logger {
	_ = sv.PrimitiveQuicServer.SetLogger(l)

	return sv.Logging.SetLogger(l)
}

func (sv *Server) SetHasSealHandler(fn network.HasSealHandler) {
	sv.hasSealHandler = fn
}

func (sv *Server) SetGetSealsHandler(fn network.GetSealsHandler) {
	sv.getSealsHandler = fn
}

func (sv *Server) SetNewSealHandler(fn network.NewSealHandler) {
	sv.newSealHandler = fn
}

func (sv *Server) SetGetManifestsHandler(fn network.GetManifestsHandler) {
	sv.getManifestsHandler = fn
}

func (sv *Server) SetGetBlocksHandler(fn network.GetBlocksHandler) {
	sv.getBlocksHandler = fn
}

func (sv *Server) NodeInfoHandler() network.NodeInfoHandler {
	return sv.nodeInfoHandler
}

func (sv *Server) SetNodeInfoHandler(fn network.NodeInfoHandler) {
	sv.nodeInfoHandler = fn
}

func (sv *Server) setHandlers() {
	_ = sv.SetHandler(QuicHandlerPathGetSeals, sv.handleGetSeals).Methods("POST")
	_ = sv.SetHandler(QuicHandlerPathSendSeal, sv.handleNewSeal).Methods("POST")
	_ = sv.SetHandler(QuicHandlerPathGetManifests, sv.handleGetManifests).Methods("POST")
	_ = sv.SetHandler(QuicHandlerPathGetBlocks, sv.handleGetBlocks).Methods("POST")
	_ = sv.SetHandler(QuicHandlerPathNodeInfo, sv.handleNodeInfo)
}

func (sv *Server) handleGetSeals(w http.ResponseWriter, r *http.Request) {
	if sv.getSealsHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		sv.Log().Error().Err(err).Msg("failed to read post body")

		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(r.Header, sv.encs, sv.enc); err != nil {
		sv.Log().Error().Err(err).Msg("failed to read encoder hint")
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else {
		enc = e
	}

	var args HashesArgs
	if err := enc.Decode(body.Bytes(), &args); err != nil {
		sv.Log().Error().Err(err).Msg("failed to decode")
		network.HTTPError(w, http.StatusBadRequest)
		return
	}

	var output []byte
	if sls, err := sv.getSealsHandler(args.Hashes); err != nil {
		sv.Log().Error().Err(err).Msg("failed to get seals")
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else if b, err := sv.enc.Marshal(sls); err != nil {
		sv.Log().Error().Err(err).Msg("failed to encode seals")
		network.HTTPError(w, http.StatusInternalServerError)
		return
	} else {
		output = b
	}

	w.Header().Set(QuicEncoderHintHeader, sv.enc.Hint().String())
	_, _ = w.Write(output)
}

func (sv *Server) handleNewSeal(w http.ResponseWriter, r *http.Request) {
	if sv.newSealHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		sv.Log().Error().Err(err).Msg("failed to read post body")

		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(r.Header, sv.encs, sv.enc); err != nil {
		network.HTTPError(w, http.StatusBadRequest)
		return
	} else {
		enc = e
	}

	var sl seal.Seal
	if s, err := seal.DecodeSeal(enc, body.Bytes()); err != nil {
		network.HTTPError(w, http.StatusBadRequest)

		sv.Log().Error().Err(err).
			Str("body", body.String()).Msg("invalid seal found")

		return
	} else {
		sl = s
	}

	// NOTE if already received, returns 200
	if sv.hasSealHandler != nil {
		if found, err := sv.hasSealHandler(sl.Hash()); err != nil {
			network.HTTPError(w, http.StatusInternalServerError)

			return
		} else if found {
			w.WriteHeader(http.StatusOK)

			return
		}
	}

	if err := sv.newSealHandler(sl); err != nil {
		seal.LoggerWithSeal(
			sl,
			sv.Log().Error().Err(err),
			sv.Log().IsVerbose(),
		).Msg("failed to receive new seal")

		network.HTTPError(w, http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (sv *Server) handleGetByHeights(
	w http.ResponseWriter,
	r *http.Request,
	getHandler func([]base.Height) (interface{}, error),
) error {
	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		sv.Log().Error().Err(err).Msg("failed to read post body")
		network.HTTPError(w, http.StatusInternalServerError)

		return err
	}

	var enc encoder.Encoder
	if e, err := EncoderFromHeader(r.Header, sv.encs, sv.enc); err != nil {
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
	} else if b, err := sv.enc.Marshal(sls); err != nil {
		network.HTTPError(w, http.StatusInternalServerError)

		return xerrors.Errorf("failed to encode: %w", err)
	} else {
		output = b
	}

	w.Header().Set(QuicEncoderHintHeader, sv.enc.Hint().String())
	_, _ = w.Write(output)

	return nil
}

func (sv *Server) handleGetManifests(w http.ResponseWriter, r *http.Request) {
	if sv.getManifestsHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	if err := sv.handleGetByHeights(
		w, r,
		func(heights []base.Height) (interface{}, error) {
			return sv.getManifestsHandler(heights)
		},
	); err != nil {
		sv.Log().Error().Err(err).Msg("failed to get manifests")
		return
	}
}

func (sv *Server) handleGetBlocks(w http.ResponseWriter, r *http.Request) {
	if sv.getBlocksHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	if err := sv.handleGetByHeights(
		w, r,
		func(heights []base.Height) (interface{}, error) {
			return sv.getBlocksHandler(heights)
		}); err != nil {
		sv.Log().Error().Err(err).Msg("failed to get blocks")
		return
	}
}

func (sv *Server) handleNodeInfo(w http.ResponseWriter, _ *http.Request) {
	if sv.nodeInfoHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)

		return
	}

	if i, err := sv.cache.Get(cacheKeyNodeInfo); err == nil {
		if output, ok := i.([]byte); ok {
			w.Header().Set(QuicEncoderHintHeader, sv.enc.Hint().String())
			_, _ = w.Write(output)

			return
		}
	}

	var output []byte
	if n, err := sv.nodeInfoHandler(); err != nil {
		sv.Log().Error().Err(err).Msg("failed to get node info")

		network.HTTPError(w, http.StatusInternalServerError)

		return
	} else if b, err := sv.enc.Marshal(n); err != nil {
		sv.Log().Error().Err(err).Msg("failed to encode node info")

		network.HTTPError(w, http.StatusInternalServerError)

		return
	} else {
		output = b
		_ = sv.cache.Set(cacheKeyNodeInfo, output, time.Second*3)
	}

	w.Header().Set(QuicEncoderHintHeader, sv.enc.Hint().String())
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
