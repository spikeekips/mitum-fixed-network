package network

import (
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type Channel interface {
	Seals([]valuehash.Hash) ([]seal.Seal, error)
	SendSeal(seal.Seal) error
}
