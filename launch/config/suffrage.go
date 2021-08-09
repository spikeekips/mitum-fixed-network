package config

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/isvalid"
)

var defaultCacheSize = 10

type Suffrage interface {
	isvalid.IsValider
	SuffrageType() string
	Nodes() []base.Address
	NumberOfActing() uint
}

type EmptySuffrage struct{}

func (EmptySuffrage) SuffrageType() string {
	return "empty-suffrage"
}

func (EmptySuffrage) Nodes() []base.Address {
	return nil
}

func (EmptySuffrage) NumberOfActing() uint {
	return 0
}

func (EmptySuffrage) IsValid([]byte) error {
	return nil
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

func (FixedSuffrage) SuffrageType() string {
	return "fixed-suffrage"
}

func (fd FixedSuffrage) Nodes() []base.Address {
	return fd.nodes
}

func (fd FixedSuffrage) NumberOfActing() uint {
	return fd.numberOfActing
}

func (fd FixedSuffrage) IsValid([]byte) error {
	switch n := uint(len(fd.nodes)); {
	case n < 1:
		return errors.Errorf("empty nodes in fixed-suffrage")
	case fd.numberOfActing < 1:
		return errors.Errorf("number-of-acting should be over zero")
	case fd.numberOfActing > n:
		return errors.Errorf("invalid number-of-acting in fixed-suffrage; over nodes")
	}

	return nil
}

type RoundrobinSuffrage struct {
	nodes          []base.Address
	numberOfActing uint
	CacheSize      int
}

func NewRoundrobinSuffrage(nodes []base.Address, numberOfActing uint) RoundrobinSuffrage {
	return RoundrobinSuffrage{nodes: nodes, numberOfActing: numberOfActing, CacheSize: defaultCacheSize}
}

func (RoundrobinSuffrage) SuffrageType() string {
	return "roundrobin"
}

func (fd RoundrobinSuffrage) Nodes() []base.Address {
	return fd.nodes
}

func (fd RoundrobinSuffrage) NumberOfActing() uint {
	return fd.numberOfActing
}

func (fd RoundrobinSuffrage) IsValid([]byte) error {
	switch n := uint(len(fd.nodes)); {
	case n < 1:
		return errors.Errorf("empty nodes in roundrobin-suffrage")
	case fd.numberOfActing < 1:
		return errors.Errorf("number-of-acting should be over zero")
	case fd.numberOfActing > n:
		return errors.Errorf("invalid number-of-acting in roundrobin-suffrage; over nodes")
	}

	return nil
}
