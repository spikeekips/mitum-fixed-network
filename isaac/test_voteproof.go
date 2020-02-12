// +build test

package isaac

import (
	"time"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/util"
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

func (vp DummyVoteProof) FinishedAt() time.Time {
	return time.Now()
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

func (vp DummyVoteProof) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		HT Height
		RD Round
		SG Stage
		RS VoteProofResultType
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(vp.Hint()),
		HT:                 vp.height,
		RD:                 vp.round,
		SG:                 vp.stage,
		RS:                 vp.result,
	})
}

func (vp *DummyVoteProof) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var uvp struct {
		HT Height
		RD Round
		SG Stage
		RS VoteProofResultType
	}

	if err := enc.Unmarshal(b, &uvp); err != nil {
		return err
	}

	vp.height = uvp.HT
	vp.round = uvp.RD
	vp.stage = uvp.SG
	vp.result = uvp.RS

	return nil
}
