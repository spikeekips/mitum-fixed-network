package config

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
)

type Node interface {
	Address() base.Address
	SetAddress(string) error
}

type RemoteNode interface {
	Node
	Publickey() key.Publickey
	SetPublickey(string) error
	ConnInfo() network.ConnInfo
	SetConnInfo(string, bool) error
}

type BaseRemoteNode struct {
	enc       encoder.Encoder
	address   base.Address
	publickey key.Publickey
	c         network.ConnInfo
}

func NewBaseRemoteNode(enc encoder.Encoder) *BaseRemoteNode {
	return &BaseRemoteNode{
		enc: enc,
	}
}

func (no BaseRemoteNode) Address() base.Address {
	return no.address
}

func (no *BaseRemoteNode) SetAddress(s string) error {
	address, err := base.DecodeAddressFromString(s, no.enc)
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

func (no BaseRemoteNode) ConnInfo() network.ConnInfo {
	return no.c
}

func (no *BaseRemoteNode) SetConnInfo(s string, insecure bool) error {
	c, err := network.NewHTTPConnInfoFromString(s, insecure)
	if err != nil {
		return err
	}

	if err := c.IsValid(nil); err != nil {
		return err
	}

	no.c = c

	return nil
}
