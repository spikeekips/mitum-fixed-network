package network

import "github.com/spikeekips/mitum/seal"

type Channel interface {
	SendSeal(seal.Seal) error // NOTE should not block
	ReceiveSeal() <-chan seal.Seal
}

type SealHandler func(seal.Seal) (seal.Seal, error)
