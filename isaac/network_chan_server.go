package isaac

import (
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

type NetworkChanServer struct {
	*logging.Logging
	*util.FunctionDaemon
	newSealHandler NewSealHandler
	ch             *NetworkChanChannel
}

func NewNetworkChanServer(ch *NetworkChanChannel) *NetworkChanServer {
	cs := &NetworkChanServer{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "network-chan-server")
		}),
		ch: ch,
	}

	cs.FunctionDaemon = util.NewFunctionDaemon(cs.run, false)

	return cs
}

func (cs *NetworkChanServer) SetLogger(l logging.Logger) logging.Logger {
	_ = cs.Logging.SetLogger(l)
	_ = cs.FunctionDaemon.SetLogger(l)

	return cs.Log()
}

func (cs *NetworkChanServer) SetGetSealsHandler(GetSealsHandler) {}

func (cs *NetworkChanServer) SetNewSealHandler(f NewSealHandler) {
	cs.newSealHandler = f
}

func (cs *NetworkChanServer) SetGetManifests(GetManifestsHandler) {}
func (cs *NetworkChanServer) SetGetBlocks(GetBlocksHandler)       {}

func (cs *NetworkChanServer) run(stopChan chan struct{}) error {
end:
	for {
		select {
		case <-stopChan:
			break end
		case sl := <-cs.ch.ReceiveSeal():
			go func(sl seal.Seal) {
				if cs.newSealHandler == nil {
					cs.Log().Error().Msg("no NewSealHandler")
					return
				}

				if err := cs.newSealHandler(sl); err != nil {
					cs.Log().Error().Err(err).Msg("failed to receive new seal")
					return
				}
			}(sl)
		}
	}

	return nil
}
