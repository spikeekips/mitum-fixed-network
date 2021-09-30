package channetwork

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type Server struct {
	*logging.Logging
	*util.ContextDaemon
	newSealHandler network.NewSealHandler
	ch             *Channel
	passthroughs   func(context.Context, network.PassthroughedSeal, func(seal.Seal, network.Channel)) error
}

func NewServer(
	ch *Channel,
	passthroughs func(context.Context, network.PassthroughedSeal, func(seal.Seal, network.Channel)) error,
) *Server {
	sv := &Server{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "network-chan-server")
		}),
		ch:           ch,
		passthroughs: passthroughs,
	}

	sv.ContextDaemon = util.NewContextDaemon("network-chan-server", sv.run)

	return sv
}

func (*Server) Initialize() error {
	return nil
}

func (sv *Server) SetLogging(l *logging.Logging) *logging.Logging {
	_ = sv.ContextDaemon.SetLogging(l)

	return sv.Logging.SetLogging(l)
}

func (*Server) SetHasSealHandler(network.HasSealHandler)   {}
func (*Server) SetGetSealsHandler(network.GetSealsHandler) {}

func (sv *Server) SetNewSealHandler(f network.NewSealHandler) {
	sv.newSealHandler = f
}

func (*Server) SetNodeInfoHandler(network.NodeInfoHandler)           {}
func (*Server) NodeInfoHandler() network.NodeInfoHandler             { return nil }
func (*Server) SetBlockDataMapsHandler(network.BlockDataMapsHandler) {}
func (*Server) SetBlockDataHandler(network.BlockDataHandler)         {}
func (*Server) SetStartHandoverHandler(network.StartHandoverHandler) {}
func (*Server) SetPingHandoverHandler(network.PingHandoverHandler)   {}
func (*Server) SetEndHandoverHandler(network.EndHandoverHandler)     {}

func (sv *Server) run(ctx context.Context) error {
end:
	for {
		select {
		case <-ctx.Done():
			break end
		case sl := <-sv.ch.ReceiveSeal():
			go func(sl network.PassthroughedSeal) {
				go func() {
					if err := sv.doPassthroughs(sl); err != nil {
						sv.Log().Error().Err(err).Msg("failed to passthroughs")
					}
				}()

				if sv.newSealHandler == nil {
					sv.Log().Error().Msg("no NewSealHandler")
					return
				}

				if err := sv.newSealHandler(sl); err != nil {
					seal.LogEventSeal(sl, "seal", sv.Log().Error(), sv.IsTraceLog()).
						Err(err).Msg("failed to receive new seal")

					return
				}
			}(sl)
		}
	}

	return nil
}

func (sv *Server) doPassthroughs(sl network.PassthroughedSeal) error {
	if sv.passthroughs == nil {
		return nil
	}

	return sv.passthroughs(
		context.Background(),
		sl,
		func(sl seal.Seal, ch network.Channel) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			if err := ch.SendSeal(ctx, sv.ch.ConnInfo(), sl); err != nil {
				sv.Log().Error().Err(err).Stringer("remote", ch.ConnInfo()).Msg("failed to send seal")
			}
		},
	)
}
