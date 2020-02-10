package network

import "github.com/spikeekips/mitum/seal"

type Channel interface {
	SendSeal(seal.Seal) error
	ReceiveSeal() <-chan seal.Seal
}

type SealHandler func(seal.Seal) (seal.Seal, error)
