package network

import "github.com/spikeekips/mitum/seal"

type ChanChannel struct {
	recvChan chan seal.Seal
}

func NewChanChannel(bufsize uint) *ChanChannel {
	return &ChanChannel{
		recvChan: make(chan seal.Seal, bufsize),
	}
}

func (gs *ChanChannel) SendSeal(sl seal.Seal) error {
	go func() {
		gs.recvChan <- sl
	}()

	return nil
}

func (gs *ChanChannel) ReceiveSeal() <-chan seal.Seal {
	return gs.recvChan
}
