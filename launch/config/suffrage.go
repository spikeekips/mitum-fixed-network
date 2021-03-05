package config

import (
	"github.com/spikeekips/mitum/base"
)

var defaultCacheSize int = 10

type Suffrage interface {
	SuffrageType() string
}

type FixedSuffrage struct {
	Proposer       base.Address
	Nodes          []base.Address
	NumberOfActing uint
	CacheSize      int
}

func NewFixedSuffrage(proposer base.Address, nodes []base.Address, numberOfActing uint) FixedSuffrage {
	return FixedSuffrage{Proposer: proposer, Nodes: nodes, NumberOfActing: numberOfActing, CacheSize: defaultCacheSize}
}

func (fd FixedSuffrage) SuffrageType() string {
	return "fixed-suffrage"
}

type RoundrobinSuffrage struct {
	Nodes          []base.Address
	NumberOfActing uint
	CacheSize      int
}

func NewRoundrobinSuffrage(nodes []base.Address, numberOfActing uint) RoundrobinSuffrage {
	return RoundrobinSuffrage{Nodes: nodes, NumberOfActing: numberOfActing, CacheSize: defaultCacheSize}
}

func (fd RoundrobinSuffrage) SuffrageType() string {
	return "roundrobin"
}
