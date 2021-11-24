package base

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
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
	Majority() BallotFact
	Facts() []BallotFact
	Votes() []SignedBallotFact
	ThresholdRatio() ThresholdRatio
	Suffrages() []Address
}

func CompareVoteproofSamePoint(a, b Voteproof) int {
	if a == nil || b == nil {
		return -1
	}

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

	return 0
}

func CompareVoteproof(a, b Voteproof) int {
	if i := CompareVoteproofSamePoint(a, b); i != 0 {
		return i
	}

	if a.Stage() > b.Stage() {
		return 1
	} else if a.Stage() < b.Stage() {
		return -1
	}

	return 0
}
