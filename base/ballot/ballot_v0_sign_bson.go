package ballot

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (sb SIGNV0) MarshalBSON() ([]byte, error) {
	m := PackBaseBallotV0BSON(sb)

	m["proposal"] = sb.proposal
	m["new_block"] = sb.newBlock

	return bsonenc.Marshal(m)
}

type SIGNV0UnpackerBSON struct {
	PR valuehash.Bytes `bson:"proposal"`
	NB valuehash.Bytes `bson:"new_block"`
}

func (sb *SIGNV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	bb, bf, err := sb.BaseBallotV0.unpackBSON(b, enc)
	if err != nil {
		return err
	}

	var nib SIGNV0UnpackerBSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	}

	return sb.unpack(enc, bb, bf, nib.PR, nib.NB)
}
