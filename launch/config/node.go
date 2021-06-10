package config

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
)

type Node interface {
	Address() base.Address
	SetAddress(string) error
}

type RemoteNode interface {
	Node
	NodeNetwork
	Publickey() key.Publickey
	SetPublickey(string) error
}

type BaseRemoteNode struct {
	*BaseNodeNetwork
	enc       encoder.Encoder
	address   base.Address
	publickey key.Publickey
}

func NewBaseRemoteNode(enc encoder.Encoder) *BaseRemoteNode {
	return &BaseRemoteNode{
		BaseNodeNetwork: EmptyBaseNodeNetwork(),
		enc:             enc,
	}
}

func (no BaseRemoteNode) Address() base.Address {
	return no.address
}

func (no *BaseRemoteNode) SetAddress(s string) error {
	address, err := base.DecodeAddressFromString(no.enc, s)
	if err != nil {
		return err
	}
	no.address = address

	return nil
}

func (no BaseRemoteNode) Publickey() key.Publickey {
	return no.publickey
}

func (no *BaseRemoteNode) SetPublickey(s string) error {
	pub, err := key.DecodePublickey(no.enc, s)
	if err != nil {
		return err
	}
	no.publickey = pub

	return nil
}
