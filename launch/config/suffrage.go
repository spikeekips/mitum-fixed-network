package config

import (
	"github.com/spikeekips/mitum/base"
)

var defaultCacheSize int = 10

type Suffrage interface {
	SuffrageType() string
	Nodes() []base.Address
	NumberOfActing() uint
}

type FixedSuffrage struct {
	Proposer       base.Address
	nodes          []base.Address
	numberOfActing uint
	CacheSize      int
}

func NewFixedSuffrage(proposer base.Address, nodes []base.Address, numberOfActing uint) FixedSuffrage {
	return FixedSuffrage{Proposer: proposer, nodes: nodes, numberOfActing: numberOfActing, CacheSize: defaultCacheSize}
}

func (fd FixedSuffrage) SuffrageType() string {
	return "fixed-suffrage"
}

func (fd FixedSuffrage) Nodes() []base.Address {
	return fd.nodes
}

func (fd FixedSuffrage) NumberOfActing() uint {
	return fd.numberOfActing
}

type RoundrobinSuffrage struct {
	nodes          []base.Address
	numberOfActing uint
	CacheSize      int
}

func NewRoundrobinSuffrage(nodes []base.Address, numberOfActing uint) RoundrobinSuffrage {
	return RoundrobinSuffrage{nodes: nodes, numberOfActing: numberOfActing, CacheSize: defaultCacheSize}
}

func (fd RoundrobinSuffrage) SuffrageType() string {
	return "roundrobin"
}

func (fd RoundrobinSuffrage) Nodes() []base.Address {
	return fd.nodes
}

func (fd RoundrobinSuffrage) NumberOfActing() uint {
	return fd.numberOfActing
}
