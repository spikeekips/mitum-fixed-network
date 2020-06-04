// +build test

package base

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/valuehash"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

func NewTestVoteproofV0(
	height Height,
	round Round,
	threshold Threshold,
	result VoteResultType,
	closed bool,
	stage Stage,
	majority Fact,
	facts map[valuehash.Hash]Fact,
	ballots map[Address]valuehash.Hash,
	votes map[Address]VoteproofNodeFact,
	finishedAt time.Time,
) VoteproofV0 {
	return VoteproofV0{
		height:     height,
		round:      round,
		threshold:  threshold,
		result:     result,
		closed:     closed,
		stage:      stage,
		majority:   majority,
		facts:      facts,
		ballots:    ballots,
		votes:      votes,
		finishedAt: finishedAt,
	}
}

var (
	DummyVoteproofType = hint.MustNewType(0xff, 0x50, "dummy-voteproof")
	DummyVoteproofHint = hint.MustHint(DummyVoteproofType, "0.1.0-dummy")
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

func (vp DummyVoteproof) Majority() Fact {
	return nil
}

func (vp DummyVoteproof) Ballots() map[Address]valuehash.Hash {
	return nil
}

func (vp DummyVoteproof) Votes() map[Address]VoteproofNodeFact {
	return nil
}

func (vp DummyVoteproof) Threshold() Threshold {
	return Threshold{}
}

func (vp DummyVoteproof) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(struct {
		jsonencoder.HintedHead
		HT Height
		RD Round
		SG Stage
		RS VoteResultType
	}{
		HintedHead: jsonencoder.NewHintedHead(vp.Hint()),
		HT:         vp.height,
		RD:         vp.round,
		SG:         vp.stage,
		RS:         vp.result,
	})
}

func (vp *DummyVoteproof) UnpackJSON(b []byte, enc *jsonencoder.Encoder) error {
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
	return bsonencoder.Marshal(bson.M{
		"_hint":  vp.Hint(),
		"height": vp.height,
		"round":  vp.round,
		"stage":  vp.stage,
		"result": vp.result,
	})
}

func (vp *DummyVoteproof) UnpackBSON(b []byte, enc *bsonencoder.Encoder) error {
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

func (vp DummyVoteproof) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	if !verbose {
		return e.Dict(key, logging.Dict().
			Hinted("height", vp.height).
			Hinted("round", vp.round).
			Hinted("stage", vp.stage).
			Str("result", vp.result.String()))
	}

	r, _ := jsonencoder.Marshal(vp)

	return e.RawJSON(key, r)
}
