package isaac

import (
	"github.com/Masterminds/semver"
	"github.com/spikeekips/mitum/common"
)

var (
	CurrentBallotVersion semver.Version = *semver.MustParse("v0.1-proto")
)

type BallotBlock struct {
	Current common.Hash
	Next    common.Hash
}

type BallotBlockState struct {
	Current []byte
	Next    []byte
}

type Ballot struct {
	Version   semver.Version
	Proposer  common.Address
	VoteStage common.VoteStage
	Vote      common.Vote
	Block     BallotBlock
	State     BallotBlockState
	Proposed  common.Time
	Round     uint64

	Transactions []common.Hash // TODO check Hash.p is 'tx'
}
