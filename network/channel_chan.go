package network

import (
	"github.com/rs/zerolog"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
)

type ChanChannel struct {
	*logging.Logger
	recvChan    chan seal.Seal
	sealHandler SealHandler
}

func NewChanChannel(bufsize uint, sealHandler SealHandler) *ChanChannel {
	return &ChanChannel{
		Logger: logging.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "chan-network")
		}),
		recvChan:    make(chan seal.Seal, bufsize),
		sealHandler: sealHandler,
	}
}

func (gs *ChanChannel) SetSealHandler(sealHandler SealHandler) {
	gs.sealHandler = sealHandler
}

func (gs *ChanChannel) SendSeal(sl seal.Seal) error {
	go func() {
		if gs.sealHandler != nil {
			if s, err := gs.sealHandler(sl); err != nil {
				gs.Log().Error().Err(err).Msg("invalid seal found")
				return
			} else {
				sl = s
			}
		}

		gs.recvChan <- sl
	}()

	return nil
}

func (gs *ChanChannel) ReceiveSeal() <-chan seal.Seal {
	return gs.recvChan
}
