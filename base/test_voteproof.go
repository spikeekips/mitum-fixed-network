//go:build test
// +build test

package base

import (
	"time"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/localtime"
	"go.mongodb.org/mongo-driver/bson"
)

func NewTestVoteproofV0(
	height Height,
	round Round,
	suffrage []Address,
	thresholdRatio ThresholdRatio,
	result VoteResultType,
	closed bool,
	stage Stage,
	majority BallotFact,
	facts []BallotFact,
	votes []SignedBallotFact,
	finishedAt time.Time,
) VoteproofV0 {
	vp := EmptyVoteproofV0()
	vp.height = height
	vp.round = round
	vp.suffrages = suffrage
	vp.thresholdRatio = thresholdRatio
	vp.result = result
	vp.closed = closed
	vp.stage = stage
	vp.majority = majority
	vp.facts = facts
	vp.votes = votes
	vp.finishedAt = finishedAt

	return vp
}

var (
	DummyVoteproofType = hint.Type("dummy-voteproof")
	DummyVoteproofHint = hint.NewHint(DummyVoteproofType, "v0.1.0-dummy")
)

type DummyVoteproof struct {
	height Height
	round  Round
	stage  Stage
	result VoteResultType
}

func NewDummyVoteproof(
	height Height, round Round, stage Stage, result VoteResultType,
) DummyVoteproof {
	return DummyVoteproof{
		height: height,
		round:  round,
		stage:  stage,
		result: result,
	}
}

func (vp DummyVoteproof) ID() string {
	return util.UUID().String()
}

func (vp DummyVoteproof) Hint() hint.Hint {
	return DummyVoteproofHint
}

func (vp DummyVoteproof) IsValid([]byte) error {
	return nil
}

func (vp DummyVoteproof) FinishedAt() time.Time {
	return localtime.UTCNow()
}

func (vp DummyVoteproof) IsFinished() bool {
	return vp.result != VoteResultNotYet
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

func (vp DummyVoteproof) Result() VoteResultType {
	return vp.result
}

func (vp DummyVoteproof) Majority() BallotFact {
	return nil
}

func (vp DummyVoteproof) Facts() []BallotFact {
	return nil
}

func (vp DummyVoteproof) Votes() []SignedBallotFact {
	return nil
}

func (vp DummyVoteproof) Suffrages() []Address {
	return nil
}

func (vp DummyVoteproof) ThresholdRatio() ThresholdRatio {
	return ThresholdRatio(100)
}

func (vp DummyVoteproof) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		HT Height
		RD Round
		SG Stage
		RS VoteResultType
	}{
		HintedHead: jsonenc.NewHintedHead(vp.Hint()),
		HT:         vp.height,
		RD:         vp.round,
		SG:         vp.stage,
		RS:         vp.result,
	})
}

func (vp *DummyVoteproof) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var uvp struct {
		HT Height
		RD Round
		SG Stage
		RS VoteResultType
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

func (vp DummyVoteproof) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bson.M{
		"_hint":  vp.Hint(),
		"height": vp.height,
		"round":  vp.round,
		"stage":  vp.stage,
		"result": vp.result,
	})
}

func (vp *DummyVoteproof) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uvp struct {
		HT Height         `bson:"height"`
		RD Round          `bson:"round"`
		SG Stage          `bson:"stage"`
		RS VoteResultType `bson:"result"`
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

func (vp DummyVoteproof) MarshalZerologObject(e *zerolog.Event) {
	e.
		Int64("height", vp.height.Int64()).
		Uint64("round", vp.round.Uint64()).
		Stringer("stage", vp.stage).
		Stringer("result", vp.result)
}
