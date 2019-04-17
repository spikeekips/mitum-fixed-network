package isaac

import (
	"github.com/Masterminds/semver"
	"github.com/spikeekips/mitum/common"
)

var (
	CurrentBallotVersion semver.Version = *semver.MustParse("0.1.0-proto")
)

type BallotBlock struct {
	Current common.Hash
	Next    common.Hash
}

type BallotBlockState struct {
	Current []byte
	Next    []byte
}

type ProposeBallot struct {
	Version    semver.Version
	Proposer   common.Address
	Block      BallotBlock
	State      BallotBlockState
	ProposedAt common.Time

	Transactions []common.Hash // NOTE check Hash.p is 'tx'
}

type VoteBallot struct {
	ProposeBallot common.Hash // NOTE ProposeBallot Hash
	VoteStage     common.VoteStage
	Vote          common.Vote
	Round         uint64
}
