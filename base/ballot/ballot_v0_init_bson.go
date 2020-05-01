package ballot // nolint

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

func (ib INITBallotV0) MarshalBSON() ([]byte, error) {
	m := PackBaseBallotV0BSON(ib)

	m["previous_block"] = ib.previousBlock
	m["previous_round"] = ib.previousRound

	if ib.voteproof != nil {
		m["voteproof"] = ib.voteproof
	}

	return bsonencoder.Marshal(m)
}

type INITBallotV0UnpackerBSON struct {
	PB bson.Raw   `bson:"previous_block"`
	PR base.Round `bson:"previous_round"`
	VR bson.Raw   `bson:"voteproof,omitempty"`
}

func (ib *INITBallotV0) UnpackBSON(b []byte, enc *bsonencoder.Encoder) error {
	bb, bf, err := ib.BaseBallotV0.unpackBSON(b, enc)
	if err != nil {
		return err
	}

	var nib INITBallotV0UnpackerBSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	}

	return ib.unpack(enc, bb, bf, nib.PB, nib.PR, nib.VR)
}

func (ibf INITBallotFactV0) MarshalBSON() ([]byte, error) {
	m := NewBaseBallotFactV0PackerBSON(ibf.BaseBallotFactV0, ibf.Hint())

	m["previous_block"] = ibf.previousBlock
	m["previous_round"] = ibf.previousRound

	return bsonencoder.Marshal(m)
}

type INITBallotFactV0UnpackerBSON struct {
	PB bson.Raw   `bson:"previous_block"`
	PR base.Round `bson:"previous_round"`
}

func (ibf *INITBallotFactV0) UnpackBSON(b []byte, enc *bsonencoder.Encoder) error {
	var err error

	var bf BaseBallotFactV0
	if bf, err = ibf.BaseBallotFactV0.unpackBSON(b, enc); err != nil {
		return err
	}

	var ubf INITBallotFactV0UnpackerBSON
	if err = enc.Unmarshal(b, &ubf); err != nil {
		return err
	}

	return ibf.unpack(enc, bf, ubf.PB, ubf.PR)
}