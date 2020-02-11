// +build test

package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/valuehash"
)

var DummyVoteProofHint hint.Hint = hint.MustHint(VoteProofType, "0.1.0-dummy")

type DummyVoteProof struct {
	height Height
	round  Round
	stage  Stage
	result VoteProofResultType
}

func NewDummyVoteProof(height Height, round Round, stage Stage, result VoteProofResultType) DummyVoteProof {
	return DummyVoteProof{
		height: height,
		round:  round,
		stage:  stage,
		result: result,
	}
}

func (vp DummyVoteProof) Hint() hint.Hint {
	return DummyVoteProofHint
}

func (vp DummyVoteProof) IsValid([]byte) error {
	return nil
}

func (vp DummyVoteProof) IsFinished() bool {
	return vp.result != VoteProofNotYet
}

func (vp DummyVoteProof) IsClosed() bool {
	return false
}

func (vp DummyVoteProof) Bytes() []byte {
	return nil
}

func (vp DummyVoteProof) Height() Height {
	return vp.height
}

func (vp DummyVoteProof) Round() Round {
	return vp.round
}

func (vp DummyVoteProof) Stage() Stage {
	return vp.stage
}

func (vp DummyVoteProof) Result() VoteProofResultType {
	return vp.result
}

func (vp DummyVoteProof) Majority() Fact {
	return nil
}

func (vp DummyVoteProof) Ballots() map[Address]valuehash.Hash {
	return nil
}

func (vp DummyVoteProof) CompareWithBlock(Block) error {
	return nil
}
