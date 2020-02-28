package network

import (
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type (
	GetSealsHandler func([]valuehash.Hash) ([]seal.Seal, error)
	NewSealHandler  func(seal.Seal) error
)

type Server interface {
	util.Daemon
	SetGetSealHandler(GetSealsHandler)
	SetNewSealHandler(NewSealHandler)
}

type Client interface {
	Send(string /* address */, []byte /* body */) error
	Request(string /* address */, []byte /* body */) (Response, error)
}

type Response interface {
	OK() bool
	Bytes() []byte
}
