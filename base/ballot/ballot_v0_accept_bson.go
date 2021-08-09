package ballot // nolint

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (ab ACCEPTV0) MarshalBSON() ([]byte, error) {
	m := PackBaseBallotV0BSON(ab)

	m["proposal"] = ab.proposal
	m["new_block"] = ab.newBlock

	if ab.voteproof != nil {
		m["voteproof"] = ab.voteproof
	}

	return bsonenc.Marshal(m)
}

type ACCEPTV0UnpackerBSON struct {
	PR valuehash.Bytes `bson:"proposal"`
	NB valuehash.Bytes `bson:"new_block"`
	VR bson.Raw        `bson:"voteproof,omitempty"`
}

func (ab *ACCEPTV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	bb, bf, err := ab.BaseBallotV0.unpackBSON(b, enc)
	if err != nil {
		return err
	}

	var nab ACCEPTV0UnpackerBSON
	if err := enc.Unmarshal(b, &nab); err != nil {
		return err
	}

	return ab.unpack(enc, bb, bf, nab.PR, nab.NB, nab.VR)
}

func (abf ACCEPTFactV0) MarshalBSON() ([]byte, error) {
	m := NewBaseBallotFactV0PackerBSON(abf.BaseFactV0, abf.Hint())

	m["proposal"] = abf.proposal
	m["new_block"] = abf.newBlock

	return bsonenc.Marshal(m)
}

type ACCEPTFactV0UnpackerBSON struct {
	PR valuehash.Bytes `bson:"proposal"`
	NB valuehash.Bytes `bson:"new_block"`
}

func (abf *ACCEPTFactV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var err error

	var bf BaseFactV0
	if bf, err = abf.BaseFactV0.unpackBSON(b, enc); err != nil {
		return err
	}

	var ubf ACCEPTFactV0UnpackerBSON
	if err = enc.Unmarshal(b, &ubf); err != nil {
		return err
	}

	return abf.unpack(enc, bf, ubf.PR, ubf.NB)
}
