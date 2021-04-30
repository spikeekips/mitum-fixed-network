package config

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/isvalid"
	"golang.org/x/xerrors"
)

var defaultCacheSize int = 10

type Suffrage interface {
	isvalid.IsValider
	SuffrageType() string
	Nodes() []base.Address
	NumberOfActing() uint
}

type EmptySuffrage struct{}

func (fd EmptySuffrage) SuffrageType() string {
	return "empty-suffrage"
}

func (fd EmptySuffrage) Nodes() []base.Address {
	return nil
}

func (fd EmptySuffrage) NumberOfActing() uint {
	return 0
}

func (fd EmptySuffrage) IsValid([]byte) error {
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

func (fd FixedSuffrage) SuffrageType() string {
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
		return xerrors.Errorf("empty nodes in fixed-suffrage")
	case fd.numberOfActing < 1:
		return xerrors.Errorf("number-of-acting should be over zero")
	case fd.numberOfActing > n:
		return xerrors.Errorf("invalid number-of-acting in fixed-suffrage; over nodes")
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

func (fd RoundrobinSuffrage) SuffrageType() string {
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
		return xerrors.Errorf("empty nodes in roundrobin-suffrage")
	case fd.numberOfActing < 1:
		return xerrors.Errorf("number-of-acting should be over zero")
	case fd.numberOfActing > n:
		return xerrors.Errorf("invalid number-of-acting in roundrobin-suffrage; over nodes")
	}

	return nil
}
