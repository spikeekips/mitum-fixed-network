// +build test

package isaac

import (
	"time"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var DummyVoteproofHint hint.Hint = hint.MustHint(VoteproofType, "0.1.0-dummy")

type DummyVoteproof struct {
	height Height
	round  Round
	stage  Stage
	result VoteproofResultType
}

func NewDummyVoteproof(height Height, round Round, stage Stage, result VoteproofResultType) DummyVoteproof {
	return DummyVoteproof{
		height: height,
		round:  round,
		stage:  stage,
		result: result,
	}
}

func (vp DummyVoteproof) Hint() hint.Hint {
	return DummyVoteproofHint
}

func (vp DummyVoteproof) IsValid([]byte) error {
	return nil
}

func (vp DummyVoteproof) FinishedAt() time.Time {
	return time.Now()
}

func (vp DummyVoteproof) IsFinished() bool {
	return vp.result != VoteproofNotYet
}

func (vp DummyVoteproof) IsClosed() bool {
	return false
}

func (vp DummyVoteproof) Bytes() []byte {
	return nil
}

func (vp DummyVoteproof) Height() Height {
	return vp.height
}

func (vp DummyVoteproof) Round() Round {
	return vp.round
}

func (vp DummyVoteproof) Stage() Stage {
	return vp.stage
}

func (vp DummyVoteproof) Result() VoteproofResultType {
	return vp.result
}

func (vp DummyVoteproof) Majority() operation.Fact {
	return nil
}

func (vp DummyVoteproof) Ballots() map[Address]valuehash.Hash {
	return nil
}

func (vp DummyVoteproof) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		HT Height
		RD Round
		SG Stage
		RS VoteproofResultType
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(vp.Hint()),
		HT:                 vp.height,
		RD:                 vp.round,
		SG:                 vp.stage,
		RS:                 vp.result,
	})
}

func (vp *DummyVoteproof) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var uvp struct {
		HT Height
		RD Round
		SG Stage
		RS VoteproofResultType
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
