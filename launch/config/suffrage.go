package config

import "github.com/spikeekips/mitum/base"

var defaultCacheSize int = 10

type Suffrage interface {
	SuffrageType() string
}

type FixedProposerSuffrage struct {
	// TODO rename to FixedSuffrage
	// TODO add node list
	Proposer base.Address
}

func NewFixedProposerSuffrage(proposer base.Address) (FixedProposerSuffrage, error) {
	return FixedProposerSuffrage{Proposer: proposer}, proposer.IsValid(nil)
}

func (fd FixedProposerSuffrage) SuffrageType() string {
	return "fixed-proposer" // TODO rename to fixed-suffrage
}

type RoundrobinSuffrage struct {
	CacheSize int
	// TODO add NumberOfActingSuffrageNodes
}

func NewRoundrobinSuffrage() RoundrobinSuffrage {
	return RoundrobinSuffrage{CacheSize: defaultCacheSize}
}

func (fd RoundrobinSuffrage) SuffrageType() string {
	return "roundrobin"
}
