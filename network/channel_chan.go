package network

import (
	"github.com/rs/zerolog"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/logging"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type ChanChannel struct {
	*logging.Logging
	recvChan       chan seal.Seal
	getSealHandler GetSealsHandler
}

func NewChanChannel(bufsize uint) *ChanChannel {
	return &ChanChannel{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "chan-network")
		}),
		recvChan: make(chan seal.Seal, bufsize),
	}
}

func (gs *ChanChannel) Seals(h []valuehash.Hash) ([]seal.Seal, error) {
	if gs.getSealHandler == nil {
		return nil, xerrors.Errorf("getSealHandler is missing")
	}

	return gs.getSealHandler(h)
}

func (gs *ChanChannel) SendSeal(sl seal.Seal) error {
	gs.recvChan <- sl

	return nil
}

func (gs *ChanChannel) ReceiveSeal() <-chan seal.Seal {
	return gs.recvChan
}

func (gs *ChanChannel) SetGetSealHandler(fn GetSealsHandler) {
	gs.getSealHandler = fn
}
