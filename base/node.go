package base

import "github.com/spikeekips/mitum/base/key"

type Node interface {
	Address() Address
	Publickey() key.Publickey
}
