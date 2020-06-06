package channetwork

import (
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type NetworkChanServer struct {
	*logging.Logging
	*util.FunctionDaemon
	newSealHandler network.NewSealHandler
	ch             *NetworkChanChannel
}

func NewNetworkChanServer(ch *NetworkChanChannel) *NetworkChanServer {
	cs := &NetworkChanServer{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
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

func (cs *NetworkChanServer) SetHasSealHandler(network.HasSealHandler)   {}
func (cs *NetworkChanServer) SetGetSealsHandler(network.GetSealsHandler) {}

func (cs *NetworkChanServer) SetNewSealHandler(f network.NewSealHandler) {
	cs.newSealHandler = f
}

func (cs *NetworkChanServer) SetGetManifestsHandler(network.GetManifestsHandler) {}
func (cs *NetworkChanServer) SetGetBlocksHandler(network.GetBlocksHandler)       {}
func (cs *NetworkChanServer) SetNodeInfoHandler(network.NodeInfoHandler)         {}

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
					seal.LoggerWithSeal(
						sl,
						cs.Log().Error().Err(err),
						cs.Log().IsVerbose(),
					).Msg("failed to receive new seal")

					return
				}
			}(sl)
		}
	}

	return nil
}
