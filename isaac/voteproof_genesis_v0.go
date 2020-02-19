package isaac

import (
	"time"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/valuehash"
)

var VoteproofGenesisV0Hint hint.Hint = hint.MustHint(VoteproofType, "0.1")

type VoteproofGenesisV0 struct {
	height     Height
	threshold  Threshold
	stage      Stage
	finishedAt time.Time
}

func NewVoteproofGenesisV0(height Height, threshold Threshold, stage Stage) VoteproofGenesisV0 {
	return VoteproofGenesisV0{
		height:     height,
		threshold:  threshold,
		stage:      stage,
		finishedAt: localtime.Now(),
	}
}

func (vpg VoteproofGenesisV0) Hint() hint.Hint {
	return VoteproofGenesisV0Hint
}

func (vpg VoteproofGenesisV0) IsFinished() bool {
	return true
}

func (vpg VoteproofGenesisV0) FinishedAt() time.Time {
	return vpg.finishedAt
}

func (vpg VoteproofGenesisV0) IsClosed() bool {
	return true
}

func (vpg VoteproofGenesisV0) Height() Height {
	return vpg.height
}

func (vpg VoteproofGenesisV0) Round() Round {
	return Round(0)
}

func (vpg VoteproofGenesisV0) Stage() Stage {
	return vpg.stage
}

func (vpg VoteproofGenesisV0) Result() VoteproofResultType {
	return VoteproofMajority
}

func (vpg VoteproofGenesisV0) Majority() Fact {
	return nil
}

func (vpg VoteproofGenesisV0) Ballots() map[Address]valuehash.Hash {
	return nil
}

func (vpg VoteproofGenesisV0) Bytes() []byte {
	return nil
}

func (vpg VoteproofGenesisV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vpg.height,
		vpg.threshold,
		vpg.stage,
	}, b, false); err != nil {
		return err
	}

	if vpg.finishedAt.IsZero() {
		return isvalid.InvalidError.Wrapf("empty finishedAt")
	}

	return nil
}
