package ballot

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

func (sb SIGNBallotV0) MarshalBSON() ([]byte, error) {
	m := PackBaseBallotV0BSON(sb)

	m["proposal"] = sb.proposal
	m["new_block"] = sb.newBlock

	return bsonencoder.Marshal(m)
}

type SIGNBallotV0UnpackerBSON struct {
	PR bson.Raw `bson:"proposal"`
	NB bson.Raw `bson:"new_block"`
}

func (sb *SIGNBallotV0) UnpackBSON(b []byte, enc *bsonencoder.Encoder) error {
	bb, bf, err := sb.BaseBallotV0.unpackBSON(b, enc)
	if err != nil {
		return err
	}

	var nib SIGNBallotV0UnpackerBSON
	if err := enc.Unmarshal(b, &nib); err != nil {
		return err
	}

	return sb.unpack(enc, bb, bf, nib.PR, nib.NB)
}
