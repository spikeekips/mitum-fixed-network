package ballot // nolint

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (ib INITV0) MarshalBSON() ([]byte, error) {
	m := PackBaseBallotV0BSON(ib)

	m["previous_block"] = ib.previousBlock

	if ib.voteproof != nil {
		m["voteproof"] = ib.voteproof
	}
	if ib.acceptVoteproof != nil {
		m["accept_voteproof"] = ib.acceptVoteproof
	}

	return bsonenc.Marshal(m)
}

type INITV0UnpackerBSON struct {
	PB  valuehash.Bytes `bson:"previous_block"`
	VR  bson.Raw        `bson:"voteproof,omitempty"`
	AVR bson.Raw        `bson:"accept_voteproof,omitempty"`
}

func (ib *INITV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	bb, bf, err := ib.BaseBallotV0.unpackBSON(b, enc)
	if err != nil {
		return err
	}

	var nib INITV0UnpackerBSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	}

	return ib.unpack(enc, bb, bf, nib.PB, nib.VR, nib.AVR)
}

func (ibf INITFactV0) MarshalBSON() ([]byte, error) {
	m := NewBaseBallotFactV0PackerBSON(ibf.BaseFactV0, ibf.Hint())

	m["previous_block"] = ibf.previousBlock

	return bsonenc.Marshal(m)
}

type INITFactV0UnpackerBSON struct {
	PB valuehash.Bytes `bson:"previous_block"`
}

func (ibf *INITFactV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var err error

	var bf BaseFactV0
	if bf, err = ibf.BaseFactV0.unpackBSON(b, enc); err != nil {
		return err
	}

	var ubf INITFactV0UnpackerBSON
	if err = enc.Unmarshal(b, &ubf); err != nil {
		return err
	}

	return ibf.unpack(enc, bf, ubf.PB)
}
