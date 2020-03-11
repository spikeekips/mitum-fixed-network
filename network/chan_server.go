package network

import (
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
)

type ChanServer struct {
	*logging.Logger
	*util.FunctionDaemon
	getSealsHandler GetSealsHandler
	newSealHandler  NewSealHandler
	ch              *ChanChannel
}

func NewChanServer(ch *ChanChannel) *ChanServer {
	cs := &ChanServer{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "network-chan-server")
		}),
		ch: ch,
	}

	cs.FunctionDaemon = util.NewFunctionDaemon(cs.run, false)

	return cs
}

func (cs *ChanServer) SetLogger(l zerolog.Logger) *logging.Logger {
	_ = cs.Logger.SetLogger(l)
	_ = cs.FunctionDaemon.SetLogger(l)

	return cs.Logger
}

func (cs *ChanServer) SetGetSealsHandler(fn GetSealsHandler) {
	cs.getSealsHandler = fn
}

func (cs *ChanServer) SetNewSealHandler(fn NewSealHandler) {
	cs.newSealHandler = fn
}

func (cs *ChanServer) run(stopChan chan struct{}) error {
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
