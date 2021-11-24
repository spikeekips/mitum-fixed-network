package ballot

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (sl BaseSeal) MarshalBSON() ([]byte, error) {
	return bson.Marshal(bsonenc.MergeBSONM(
		sl.BaseSeal.BSONPacker(),
		bson.M{
			"signed_fact":      sl.sfs,
			"base_voteproof":   sl.baseVoteproof,
			"accept_voteproof": sl.acceptVoteproof,
		}))
}

type BaseBallotUnpackerBSON struct {
	F  bson.Raw `bson:"signed_fact"`
	BB bson.Raw `bson:"base_voteproof"`
	BA bson.Raw `bson:"accept_voteproof,omitempty"`
}

func (sl *BaseSeal) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	if err := sl.BaseSeal.UnpackBSON(b, enc); err != nil {
		return err
	}

	var ub BaseBallotUnpackerBSON
	if err := enc.Unmarshal(b, &ub); err != nil {
		return err
	}

	return sl.unpack(enc, ub.F, ub.BB, ub.BA)
}
