package quicnetwork

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
	"golang.org/x/xerrors"

	"github.com/gorilla/mux"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/errors"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	DefaultPort                        = "54321"
	QuicHandlerPathGetSeals            = "/seals"
	QuicHandlerPathSendSeal            = "/seal"
	QuicHandlerPathGetBlockDataMaps    = "/blockdatamaps"
	QuicHandlerPathGetBlockData        = "/blockdata"
	QuicHandlerPathGetBlockDataPattern = QuicHandlerPathGetBlockData + "/{path:.*}"
	QuicHandlerPathNodeInfo            = "/"
)

var (
	BadRequestError   = errors.NewError("bad request")
	NotSupportedErorr = errors.NewError("not supported")
)

var LimitRequestByHeights = 20 // max number of reqeust heights

var cacheKeyNodeInfo = [2]byte{0x00, 0x00}

type Server struct {
	*logging.Logging
	*PrimitiveQuicServer
	encs                 *encoder.Encoders
	enc                  encoder.Encoder // NOTE default encoder.Encoder
	getSealsHandler      network.GetSealsHandler
	hasSealHandler       network.HasSealHandler
	newSealHandler       network.NewSealHandler
	nodeInfoHandler      network.NodeInfoHandler
	blockDataMapsHandler network.BlockDataMapsHandler
	blockDataHandler     network.BlockDataHandler
	cache                cache.Cache
	rg                   *singleflight.Group
}

func NewServer(
	prim *PrimitiveQuicServer,
	encs *encoder.Encoders, enc encoder.Encoder,
	ca cache.Cache,
) (*Server, error) {
	if ca == nil {
		ca = cache.Dummy{}
	}

	nqs := &Server{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "network-quic-server")
		}),
		PrimitiveQuicServer: prim,
		encs:                encs,
		enc:                 enc,
		cache:               ca,
		rg:                  &singleflight.Group{},
	}
	nqs.setHandlers()

	return nqs, nil
}

func (*Server) Initialize() error {
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

func (sv *Server) NodeInfoHandler() network.NodeInfoHandler {
	return sv.nodeInfoHandler
}

func (sv *Server) SetNodeInfoHandler(fn network.NodeInfoHandler) {
	sv.nodeInfoHandler = fn
}

func (sv *Server) SetBlockDataMapsHandler(fn network.BlockDataMapsHandler) {
	sv.blockDataMapsHandler = fn
}

func (sv *Server) SetBlockDataHandler(fn network.BlockDataHandler) {
	sv.blockDataHandler = fn
}

func (sv *Server) setHandlers() {
	_ = sv.SetHandlerFunc(QuicHandlerPathGetSeals, sv.handleGetSeals).Methods("POST")
	_ = sv.SetHandlerFunc(QuicHandlerPathSendSeal, sv.handleNewSeal).Methods("POST")
	_ = sv.SetHandlerFunc(QuicHandlerPathGetBlockDataMaps, sv.handleGetBlockDataMaps).Methods("POST")
	_ = sv.SetHandlerFunc(QuicHandlerPathGetBlockDataPattern, sv.handleGetBlockData).Methods("GET")
	_ = sv.SetHandlerFunc(QuicHandlerPathNodeInfo, sv.handleNodeInfo)
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

	enc, err := EncoderFromHeader(r.Header, sv.encs, sv.enc)
	if err != nil {
		sv.Log().Error().Err(err).Msg("failed to read encoder hint")
		network.HTTPError(w, http.StatusBadRequest)
		return
	}

	var args HashesArgs
	switch err := enc.Decode(body.Bytes(), &args); {
	case err != nil:
		sv.Log().Error().Err(err).Msg("failed to decode")
		network.HTTPError(w, http.StatusBadRequest)
		return
	case len(args.Hashes) < 1:
		network.HTTPError(w, http.StatusBadRequest)
		return
	default:
		args.Sort()
	}

	if v, err, _ := sv.rg.Do("GetSeals-"+args.String(), func() (interface{}, error) {
		i, err := sv.getSealsHandler(args.Hashes)
		if err != nil {
			return nil, err
		}
		return sv.enc.Marshal(i)
	}); err != nil {
		sv.Log().Error().Err(err).Msg("failed to get seals")

		handleError(w, err)
	} else {
		w.Header().Set(QuicEncoderHintHeader, sv.enc.Hint().String())
		_, _ = w.Write(v.([]byte))
	}
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

	enc, err := EncoderFromHeader(r.Header, sv.encs, sv.enc)
	if err != nil {
		network.HTTPError(w, http.StatusBadRequest)
		return
	}

	sl, err := seal.DecodeSeal(enc, body.Bytes())
	if err != nil {
		network.HTTPError(w, http.StatusBadRequest)

		sv.Log().Error().Err(err).
			Str("body", body.String()).Msg("invalid seal found")

		return
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
		seal.LogEventWithSeal(
			sl,
			sv.Log().Error().Err(err),
			sv.Log().IsVerbose(),
		).Msg("failed to receive new seal")

		network.HTTPError(w, http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)
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

	if v, err, shared := sv.rg.Do("NodeInfo", func() (interface{}, error) {
		i, err := sv.nodeInfoHandler()
		if err != nil {
			return nil, err
		}
		return sv.enc.Marshal(i)
	}); err != nil {
		sv.Log().Error().Err(err).Msg("failed to get node info")

		handleError(w, err)
	} else {
		if !shared {
			_ = sv.cache.Set(cacheKeyNodeInfo, v.([]byte), time.Second*3)
		}

		w.Header().Set(QuicEncoderHintHeader, sv.enc.Hint().String())
		_, _ = w.Write(v.([]byte))
	}
}

func (sv *Server) handleGetBlockDataMaps(w http.ResponseWriter, r *http.Request) {
	if sv.blockDataMapsHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)

		return
	}

	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		sv.Log().Error().Err(err).Msg("failed to read post body")
		network.HTTPError(w, http.StatusInternalServerError)

		return
	}

	enc, err := EncoderFromHeader(r.Header, sv.encs, sv.enc)
	if err != nil {
		network.HTTPError(w, http.StatusBadRequest)

		return
	}

	var args HeightsArgs
	switch err := enc.Decode(body.Bytes(), &args); {
	case err != nil:
		network.HTTPError(w, http.StatusBadRequest)

		return
	case len(args.Heights) > LimitRequestByHeights:
		network.HTTPError(w, http.StatusBadRequest)

		return
	case len(args.Heights) < 1:
		network.HTTPError(w, http.StatusBadRequest)
		return
	default:
		args.Sort()
	}

	if v, err, _ := sv.rg.Do("GetBlockDataMaps-"+args.String(), func() (interface{}, error) {
		sls, err := sv.blockDataMapsHandler(args.Heights)
		if err != nil {
			return nil, err
		}
		return sv.enc.Marshal(sls)
	}); err != nil {
		handleError(w, err)
	} else {
		w.Header().Set(QuicEncoderHintHeader, sv.enc.Hint().String())
		_, _ = w.Write(v.([]byte))
	}
}

func (sv *Server) handleGetBlockData(w http.ResponseWriter, r *http.Request) {
	if sv.blockDataHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)

		return
	}

	vars := mux.Vars(r)

	var p string
	i, found := vars["path"]
	if !found {
		network.HTTPError(w, http.StatusBadRequest)

		return
	}
	p = strings.TrimSpace(i)
	if len(p) < 1 {
		network.HTTPError(w, http.StatusBadRequest)
		return
	}

	if v, err, _ := sv.rg.Do("GetBlockData-"+p, func() (interface{}, error) {
		j, closefunc, err := sv.blockDataHandler("/" + vars["path"])
		if err != nil {
			return nil, err
		}
		return []interface{}{j, closefunc}, nil
	}); err != nil {
		handleError(w, err)
	} else {
		var j io.Reader
		var closefunc func() error
		{
			l := v.([]interface{})
			if l[0] != nil {
				j = l[0].(io.Reader)
			}

			if l[1] != nil {
				closefunc = l[1].(func() error)
			}
		}

		if closefunc != nil {
			defer func() {
				_ = closefunc()
			}()
		}

		if j == nil {
			network.HTTPError(w, http.StatusInternalServerError)
		} else if _, err := io.Copy(w, j); err != nil {
			network.HTTPError(w, http.StatusInternalServerError)
		}
	}
}

func mustQuicURL(u, p string) (string, *url.URL) {
	uu, err := url.Parse(u)
	if err != nil {
		panic(xerrors.Errorf("failed to join quic url: %w", err))
	}

	uu.Path = path.Join(uu.Path, p)

	return uu.String(), uu
}

func handleError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case xerrors.Is(err, util.NotFoundError):
		status = http.StatusNotFound
	case xerrors.Is(err, BadRequestError):
		status = http.StatusBadRequest
	}

	network.HTTPError(w, status)
}
