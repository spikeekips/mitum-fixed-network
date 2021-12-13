package quicnetwork

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/cache"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/sync/singleflight"
)

var (
	DefaultPort                         = "54321"
	QuicHandlerPathGetStagedOperations  = "/operations"
	QuicHandlerPathSendSeal             = "/seal"
	QuicHandlerPathGetProposal          = "/proposal"
	QuicHandlerPathGetProposalPattern   = "/proposal" + "/{hash:.*}"
	QuicHandlerPathGetBlockDataMaps     = "/blockdatamaps"
	QuicHandlerPathGetBlockData         = "/blockdata"
	QuicHandlerPathGetBlockDataPattern  = QuicHandlerPathGetBlockData + "/{path:.*}"
	QuicHandlerPathPingHandoverPattern  = "/handover"
	QuicHandlerPathStartHandoverPattern = QuicHandlerPathPingHandoverPattern + "/start"
	QuicHandlerPathEndHandoverPattern   = QuicHandlerPathPingHandoverPattern + "/end"
	QuicHandlerPathNodeInfo             = "/"
)

var (
	BadRequestError   = util.NewError("bad request")
	NotSupportedErorr = util.NewError("not supported")
)

var LimitRequestByHeights = 20 // max number of reqeust heights

var cacheKeyNodeInfo = [2]byte{0x00, 0x00}

const (
	SendSealFromConnInfoHeader string = "X-MITUM-FROM-CONNINFO"
)

type Server struct {
	*logging.Logging
	*PrimitiveQuicServer
	encs                       *encoder.Encoders
	enc                        encoder.Encoder // NOTE default encoder.Encoder
	getStagedOperationsHandler network.GetStagedOperationsHandler
	newSealHandler             network.NewSealHandler
	getProposalHandler         network.GetProposalHandler
	nodeInfoHandler            network.NodeInfoHandler
	blockDataMapsHandler       network.BlockDataMapsHandler
	blockDataHandler           network.BlockDataHandler
	startHandoverHandler       network.StartHandoverHandler
	pingHandoverHandler        network.PingHandoverHandler
	endHandoverHandler         network.EndHandoverHandler
	cache                      cache.Cache
	rg                         *singleflight.Group
	connInfo                   network.ConnInfo
	passthroughs               func(context.Context, network.PassthroughedSeal, func(seal.Seal, network.Channel)) error
}

func NewServer(
	prim *PrimitiveQuicServer,
	encs *encoder.Encoders, enc encoder.Encoder,
	ca cache.Cache,
	connInfo network.ConnInfo,
	passthroughs func(context.Context, network.PassthroughedSeal, func(seal.Seal, network.Channel)) error,
) (*Server, error) {
	if ca == nil {
		ca = cache.Dummy{}
	}

	nqs := &Server{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "network-quic-server")
		}),
		PrimitiveQuicServer: prim,
		encs:                encs,
		enc:                 enc,
		cache:               ca,
		rg:                  &singleflight.Group{},
		connInfo:            connInfo,
		passthroughs:        passthroughs,
	}
	nqs.setHandlers()

	return nqs, nil
}

func (*Server) Initialize() error {
	return nil
}

func (sv *Server) Start() error {
	sv.logNilHanders()

	return sv.PrimitiveQuicServer.Start()
}

func (sv *Server) SetLogging(l *logging.Logging) *logging.Logging {
	_ = sv.PrimitiveQuicServer.SetLogging(l)

	return sv.Logging.SetLogging(l)
}

func (sv *Server) Encoders() *encoder.Encoders {
	return sv.encs
}

func (sv *Server) Encoder() encoder.Encoder {
	return sv.enc
}

func (sv *Server) SetGetStagedOperationsHandler(fn network.GetStagedOperationsHandler) {
	sv.getStagedOperationsHandler = fn
}

func (sv *Server) SetNewSealHandler(fn network.NewSealHandler) {
	sv.newSealHandler = fn
}

func (sv *Server) SetGetProposalHandler(fn network.GetProposalHandler) {
	sv.getProposalHandler = fn
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

func (sv *Server) SetStartHandoverHandler(fn network.StartHandoverHandler) {
	sv.startHandoverHandler = fn
}

func (sv *Server) SetPingHandoverHandler(fn network.PingHandoverHandler) {
	sv.pingHandoverHandler = fn
}

func (sv *Server) SetEndHandoverHandler(fn network.EndHandoverHandler) {
	sv.endHandoverHandler = fn
}

func (sv *Server) setHandlers() {
	_ = sv.SetHandlerFunc(QuicHandlerPathGetStagedOperations, sv.handleGetStagedOperations).Methods("POST")
	_ = sv.SetHandlerFunc(QuicHandlerPathSendSeal, sv.handleNewSeal).Methods("POST")
	_ = sv.SetHandlerFunc(QuicHandlerPathGetProposalPattern, sv.handleGetProposal).Methods("GET")
	_ = sv.SetHandlerFunc(QuicHandlerPathGetBlockDataMaps, sv.handleGetBlockDataMaps).Methods("POST")
	_ = sv.SetHandlerFunc(QuicHandlerPathGetBlockDataPattern, sv.handleGetBlockData).Methods("GET")
	_ = sv.SetHandlerFunc(QuicHandlerPathNodeInfo, sv.handleNodeInfo)
	_ = sv.SetHandlerFunc(QuicHandlerPathPingHandoverPattern, sv.handlePingHandover)
	_ = sv.SetHandlerFunc(QuicHandlerPathStartHandoverPattern, sv.handleStartHandover)
	_ = sv.SetHandlerFunc(QuicHandlerPathEndHandoverPattern, sv.handleEndHandover)
}

func (sv *Server) handleGetStagedOperations(w http.ResponseWriter, r *http.Request) {
	if sv.getStagedOperationsHandler == nil {
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
	switch err := enc.Unmarshal(body.Bytes(), &args); {
	case err != nil:
		sv.Log().Error().Err(err).Msg("failed to decode")
		network.HTTPError(w, http.StatusBadRequest)
		return
	case len(args.Hashes) < 1:
		sv.Log().Error().Msg("empty hashes")
		network.HTTPError(w, http.StatusBadRequest)
		return
	default:
		args.Sort()
	}

	if v, err, _ := sv.rg.Do("GetStagedOperations-"+args.String(), func() (interface{}, error) {
		i, err := sv.getStagedOperationsHandler(args.Hashes)
		if err != nil {
			return nil, err
		}
		return sv.enc.Marshal(i)
	}); err != nil {
		sv.Log().Error().Interface("hashes", args.Hashes).Err(err).Msg("failed to get operationss")

		handleError(w, err)
	} else {
		w.Header().Set(QuicEncoderHintHeader, sv.enc.Hint().String())
		_, _ = w.Write(v.([]byte))
	}
}

func (sv *Server) handleNewSeal(w http.ResponseWriter, r *http.Request) {
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

	var sl seal.Seal
	if err := encoder.Decode(body.Bytes(), enc, &sl); err != nil {
		sv.Log().Error().Err(err).Stringer("body", body).Msg("invalid seal found")

		network.HTTPError(w, http.StatusBadRequest)

		return
	}

	go func() {
		if err := sv.doPassthroughs(r, sl); err != nil {
			sv.Log().Error().Err(err).Msg("failed to passthroughs")
		}
	}()

	if sv.newSealHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	if err := sv.newSealHandler(sl); err != nil {
		seal.LogEventSeal(sl, "seal", sv.Log().Error(), sv.IsTraceLog()).
			Err(err).Msg("failed to receive new seal")

		network.HTTPError(w, http.StatusInternalServerError)

		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (sv *Server) handleGetProposal(w http.ResponseWriter, r *http.Request) {
	if sv.getProposalHandler == nil {
		network.HTTPError(w, http.StatusInternalServerError)
		return
	}

	vars := mux.Vars(r)

	i, found := vars["hash"]
	if !found {
		network.HTTPError(w, http.StatusBadRequest)

		return
	}
	h := valuehash.NewBytesFromString(strings.TrimSpace(i))
	if err := h.IsValid(nil); err != nil {
		network.HTTPError(w, http.StatusBadRequest)

		return
	}

	v, err, _ := sv.rg.Do("GetPropossal-"+h.String(), func() (interface{}, error) {
		switch i, err := sv.getProposalHandler(h); {
		case err != nil:
			return nil, err
		case i == nil:
			return nil, nil
		default:
			return sv.enc.Marshal(i)
		}
	})

	if err != nil {
		sv.Log().Error().Stringer("proposal", h).Err(err).Msg("failed to get proposal")

		handleError(w, err)
	}

	if v == nil {
		network.HTTPError(w, http.StatusNotFound)

		return
	}

	w.Header().Set(QuicEncoderHintHeader, sv.enc.Hint().String())
	_, _ = w.Write(v.([]byte))
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
	switch err := enc.Unmarshal(body.Bytes(), &args); {
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
		sv.Log().Error().Err(err).Interface("heights", args.Heights).Msg("failed to get block data maps")

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

	v, err, _ := sv.rg.Do("GetBlockData-"+p, func() (interface{}, error) {
		j, closefunc, err := sv.blockDataHandler("/" + vars["path"])
		defer func() {
			_ = closefunc()
		}()

		if err != nil {
			return nil, err
		}

		b, err := ioutil.ReadAll(j)
		if err != nil {
			return nil, fmt.Errorf("failed to get block data; failed to copy: %w", err)
		}

		return b, nil
	})
	if err != nil {
		sv.Log().Error().Err(err).Str("path", p).Msg("failed to get block data")

		handleError(w, err)
	}

	if v == nil {
		sv.Log().Error().Msg("failed to get block data; empty data")

		network.HTTPError(w, http.StatusInternalServerError)
	}

	_, _ = w.Write(v.([]byte))
}

func (sv *Server) handleStartHandover(w http.ResponseWriter, r *http.Request) {
	sl, ok := sv.loadHandoverSeal(w, r)
	if !ok {
		return
	}

	i, err, _ := sv.rg.Do("handover", func() (interface{}, error) {
		return sv.startHandoverHandler(sl)
	})

	sv.handleHandoverError(w, i.(bool), err)
}

func (sv *Server) handlePingHandover(w http.ResponseWriter, r *http.Request) {
	sl, ok := sv.loadHandoverSeal(w, r)
	if !ok {
		return
	}

	i, err, _ := sv.rg.Do("handover", func() (interface{}, error) {
		return sv.pingHandoverHandler(sl)
	})

	sv.handleHandoverError(w, i.(bool), err)
}

func (sv *Server) handleEndHandover(w http.ResponseWriter, r *http.Request) {
	sl, ok := sv.loadHandoverSeal(w, r)
	if !ok {
		return
	}

	i, err, _ := sv.rg.Do("handover", func() (interface{}, error) {
		return sv.endHandoverHandler(sl)
	})

	sv.handleHandoverError(w, i.(bool), err)
}

func (sv *Server) loadHandoverSeal(w http.ResponseWriter, r *http.Request) (network.HandoverSeal, bool) {
	body := &bytes.Buffer{}
	if _, err := io.Copy(body, r.Body); err != nil {
		sv.Log().Error().Err(err).Msg("failed to read post body")

		network.HTTPError(w, http.StatusInternalServerError)
		return nil, false
	}

	enc, err := EncoderFromHeader(r.Header, sv.encs, sv.enc)
	if err != nil {
		network.HTTPError(w, http.StatusBadRequest)
		return nil, false
	}

	var sl network.HandoverSeal
	if err := encoder.Decode(body.Bytes(), enc, &sl); err != nil {
		sv.Log().Error().Err(err).Stringer("body", body).Msg("invalid handover seal found")

		network.HTTPError(w, http.StatusBadRequest)

		return nil, false
	}

	return sl, true
}

func (*Server) handleHandoverError(w http.ResponseWriter, ok bool, err error) {
	switch {
	case errors.Is(err, network.HandoverRejectedError):
		network.WriteProblemWithError(w, http.StatusNotAcceptable, err)
	case err != nil:
		network.WriteProblemWithError(w, http.StatusInternalServerError, err)
	case !ok:
		w.WriteHeader(http.StatusNotAcceptable)
	default:
		w.WriteHeader(http.StatusOK)
	}
}

func (sv *Server) logNilHanders() {
	handlers := [][2]interface{}{
		{sv.getStagedOperationsHandler, "getStagedOperationsHandler"},
		{sv.newSealHandler, "newSealHandler"},
		{sv.nodeInfoHandler, "nodeInfoHandler"},
		{sv.blockDataMapsHandler, "blockDataMapsHandler"},
		{sv.blockDataHandler, "blockDataHandler"},
	}

	var enables, disables []string
	for i := range handlers {
		f, name := handlers[i][0], handlers[i][1]

		if reflect.ValueOf(f).IsNil() {
			disables = append(disables, name.(string))
		} else {
			enables = append(enables, name.(string))
		}
	}

	sv.Log().Debug().Strs("enabled", enables).Strs("disabled", disables).Msg("check handler")
}

func (sv *Server) doPassthroughs(r *http.Request, sl seal.Seal) error {
	if sv.passthroughs == nil {
		return nil
	}

	return sv.passthroughs(
		context.Background(),
		network.NewPassthroughedSeal(sl, strings.TrimSpace(r.Header.Get(SendSealFromConnInfoHeader))),
		func(sl seal.Seal, ch network.Channel) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			l := sv.Log().With().
				Stringer("remote", ch.ConnInfo()).
				Stringer("seal", sl.Hash()).
				Logger()

			if err := ch.SendSeal(ctx, sv.connInfo, sl); err != nil {
				l.Trace().Err(err).Msg("failed to passthrough seal")

				return
			}
			l.Trace().Msg("passthroughed")
		},
	)
}

func mustQuicURL(u, p string) (string, *url.URL) {
	uu, err := network.ParseURL(u, false)
	if err != nil {
		panic(errors.Wrap(err, "failed to join quic url"))
	}

	uu.Path = path.Join(uu.Path, p)

	return uu.String(), uu
}

func handleError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, util.NotFoundError) {
		status = http.StatusNotFound
	}

	network.HTTPError(w, status)
}
