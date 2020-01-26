package mitum

import (
	"encoding/json"

	"github.com/spikeekips/mitum/errors"
)

var (
	InvalidVoteResultTypeError = errors.NewError("invalid VoteResultType")
)

type VoteResultType uint8

const (
	VoteResultNotYet VoteResultType = iota
	VoteResultDraw
	VoteResultMajority
)

func (vrt VoteResultType) String() string {
	switch vrt {
	case VoteResultNotYet:
		return "NOT-YET"
	case VoteResultDraw:
		return "DRAW"
	case VoteResultMajority:
		return "MAJORITY"
	default:
		return "<unknown VoteResultType>"
	}
}

func (vrt VoteResultType) IsValid([]byte) error {
	switch vrt {
	case VoteResultNotYet, VoteResultDraw, VoteResultMajority:
		return nil
	}

	return InvalidVoteResultTypeError.Wrapf("VoteResultType=%d", vrt)
}

func (vrt VoteResultType) MarshalText() ([]byte, error) {
	return []byte(vrt.String()), nil
}

type VoteResult struct {
	height   Height
	round    Round
	stage    Stage
	result   VoteResultType
	majority VoteRecord
	votes    map[Address]VoteRecord // key: node Address, value: VoteRecord
}

func NewVoteResult(ballot Ballot) VoteResult {
	return VoteResult{
		height: ballot.Height(),
		round:  ballot.Round(),
		stage:  ballot.Stage(),
		result: VoteResultNotYet,
		votes:  nil,
	}
}

func (vr VoteResult) String() string {
	b, _ := json.Marshal(vr)

	return string(b)
}

func (vr VoteResult) Height() Height {
	return vr.height
}

func (vr VoteResult) Round() Round {
	return vr.round
}

func (vr VoteResult) Stage() Stage {
	return vr.stage
}

func (vr VoteResult) Bytes() []byte {
	return nil
}

func (vr VoteResult) Result() VoteResultType {
	return vr.result
}

func (vr VoteResult) Majority() VoteRecord {
	return vr.majority
}

func (vr VoteResult) IsVoted(node Address) bool {
	_, found := vr.votes[node]

	return found
}

func (vr VoteResult) VoteCount() int {
	return len(vr.votes)
}

func (vr VoteResult) IsFinished() bool {
	return vr.result != VoteResultNotYet
}
