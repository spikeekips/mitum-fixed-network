package isaac

import (
	"time"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/valuehash"
)

var VoteProofGenesisV0Hint hint.Hint = hint.MustHint(VoteProofType, "0.1")

type VoteProofGenesisV0 struct {
	height     Height
	threshold  Threshold
	stage      Stage
	finishedAt time.Time
}

func NewVoteProofGenesisV0(height Height, threshold Threshold, stage Stage) VoteProofGenesisV0 {
	return VoteProofGenesisV0{
		height:     height,
		threshold:  threshold,
		stage:      stage,
		finishedAt: localtime.Now(),
	}
}

func (vpg VoteProofGenesisV0) Hint() hint.Hint {
	return VoteProofGenesisV0Hint
}

func (vpg VoteProofGenesisV0) IsFinished() bool {
	return true
}

func (vpg VoteProofGenesisV0) FinishedAt() time.Time {
	return vpg.finishedAt
}

func (vpg VoteProofGenesisV0) IsClosed() bool {
	return true
}

func (vpg VoteProofGenesisV0) Height() Height {
	return vpg.height
}

func (vpg VoteProofGenesisV0) Round() Round {
	return Round(0)
}

func (vpg VoteProofGenesisV0) Stage() Stage {
	return vpg.stage
}

func (vpg VoteProofGenesisV0) Result() VoteProofResultType {
	return VoteProofMajority
}

func (vpg VoteProofGenesisV0) Majority() Fact {
	return nil
}

func (vpg VoteProofGenesisV0) Ballots() map[Address]valuehash.Hash {
	return nil
}

func (vpg VoteProofGenesisV0) Bytes() []byte {
	return nil
}

func (vpg VoteProofGenesisV0) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vpg.height,
		vpg.threshold,
		vpg.stage,
	}, b); err != nil {
		return err
	}

	if vpg.finishedAt.IsZero() {
		return isvalid.InvalidError.Wrapf("empty finishedAt")
	}

	return nil
}

func (vpg VoteProofGenesisV0) CompareWithBlock(Block) error {
	return nil
}
