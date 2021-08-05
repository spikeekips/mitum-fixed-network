package base

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

type Voteproof interface {
	hint.Hinter
	isvalid.IsValider
	util.Byter
	zerolog.LogObjectMarshaler
	ID() string // ID is only unique in local machine
	IsFinished() bool
	FinishedAt() time.Time
	IsClosed() bool
	Height() Height
	Round() Round
	Stage() Stage
	Result() VoteResultType
	Majority() Fact
	Facts() []Fact
	Votes() []VoteproofNodeFact
	ThresholdRatio() ThresholdRatio
	Suffrages() []Address
}

type VoteproofNodeFact interface {
	hint.Hinter
	isvalid.IsValider
	util.Byter
	Ballot() valuehash.Hash
	Fact() valuehash.Hash
	Signature() key.Signature
	Node() Address
	Signer() key.Publickey
}

type VoteproofCallbacker struct {
	Voteproof
	callback func() error
}

func VoteproofWithCallback(voteproof Voteproof, callback func() error) VoteproofCallbacker {
	return VoteproofCallbacker{Voteproof: voteproof, callback: callback}
}

func (vc VoteproofCallbacker) Callback() error {
	return vc.callback()
}

type Voteproofer interface {
	Voteproof() Voteproof
}

func CompareVoteproof(a, b Voteproof) int {
	if a.Height() > b.Height() {
		return 1
	} else if a.Height() < b.Height() {
		return -1
	}

	if a.Round() > b.Round() {
		return 1
	} else if a.Round() < b.Round() {
		return -1
	}

	if a.Stage() > b.Stage() {
		return 1
	} else if a.Stage() < b.Stage() {
		return -1
	}

	return 0
}
