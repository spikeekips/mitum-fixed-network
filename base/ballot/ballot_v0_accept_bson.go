package ballot // nolint

import (
	"github.com/spikeekips/mitum/util/encoder"
	"go.mongodb.org/mongo-driver/bson"
)

func (ab ACCEPTBallotV0) MarshalBSON() ([]byte, error) {
	m := PackBaseBallotV0BSON(ab)

	m["proposal"] = ab.proposal
	m["new_block"] = ab.newBlock

	if ab.voteproof != nil {
		m["voteproof"] = ab.voteproof
	}

	return bson.Marshal(m)
}

type ACCEPTBallotV0UnpackerBSON struct {
	PR bson.Raw `bson:"proposal"`
	NB bson.Raw `bson:"new_block"`
	VR bson.Raw `bson:"voteproof,omitempty"`
}

func (ab *ACCEPTBallotV0) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error { // nolint
	bb, bf, err := ab.BaseBallotV0.unpackBSON(b, enc)
	if err != nil {
		return err
	}

	var nab ACCEPTBallotV0UnpackerBSON
	if err := enc.Unmarshal(b, &nab); err != nil {
		return err
	}

	return ab.unpack(enc, bb, bf, nab.PR, nab.NB, nab.VR)
}

func (abf ACCEPTBallotFactV0) MarshalBSON() ([]byte, error) {
	m := NewBaseBallotFactV0PackerBSON(abf.BaseBallotFactV0, abf.Hint())

	m["proposal"] = abf.proposal
	m["new_block"] = abf.newBlock

	return bson.Marshal(m)
}

type ACCEPTBallotFactV0UnpackerBSON struct {
	PR bson.Raw `bson:"proposal"`
	NB bson.Raw `bson:"new_block"`
}

func (abf *ACCEPTBallotFactV0) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
	var err error

	var bf BaseBallotFactV0
	if bf, err = abf.BaseBallotFactV0.unpackBSON(b, enc); err != nil {
		return err
	}

	var ubf ACCEPTBallotFactV0UnpackerBSON
	if err = enc.Unmarshal(b, &ubf); err != nil {
		return err
	}

	return abf.unpack(enc, bf, ubf.PR, ubf.NB)
}
