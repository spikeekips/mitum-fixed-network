//go:build test
// +build test

package base

import (
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

type baseDummyBallotFact struct {
	H  valuehash.Hash
	S  Stage
	HT Height
	R  Round
}

type DummyBallotFact struct {
	baseDummyBallotFact
	Err error
}

func NewDummyBallotFact() DummyBallotFact {
	return DummyBallotFact{
		baseDummyBallotFact: baseDummyBallotFact{
			H: valuehash.RandomSHA256(),
		},
	}
}

func (fact DummyBallotFact) Hint() hint.Hint {
	return hint.NewHint(hint.Type("dummy-ballot-fact"), "v0.0.1")
}

func (fact DummyBallotFact) Bytes() []byte {
	return nil
}

func (fact DummyBallotFact) IsValid([]byte) error {
	return fact.Err
}

func (fact DummyBallotFact) Hash() valuehash.Hash {
	return fact.H
}

func (fact DummyBallotFact) Stage() Stage {
	return fact.S
}

func (fact DummyBallotFact) Height() Height {
	return fact.HT
}

func (fact DummyBallotFact) Round() Round {
	return fact.R
}

func (fact DummyBallotFact) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		baseDummyBallotFact
	}{
		HintedHead:          jsonenc.NewHintedHead(fact.Hint()),
		baseDummyBallotFact: fact.baseDummyBallotFact,
	})
}

func (fact DummyBallotFact) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(fact.Hint()),
		bson.M{
			"H":  fact.H,
			"S":  fact.S,
			"HT": fact.HT,
			"R":  fact.R,
		},
	))
}

func (fact *DummyBallotFact) unpack(b []byte, enc encoder.Encoder) error {
	var uf struct {
		H  valuehash.Bytes
		S  Stage
		HT Height
		R  Round
	}

	if err := enc.Unmarshal(b, &uf); err != nil {
		return err
	}

	fact.H = uf.H
	fact.S = uf.S
	fact.HT = uf.HT
	fact.R = uf.R

	return nil
}

func (fact *DummyBallotFact) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	return fact.unpack(b, enc)
}

func (fact *DummyBallotFact) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	return fact.unpack(b, enc)
}
