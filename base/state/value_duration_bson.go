package state

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (dv DurationValue) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(dv.Hint()),
		bson.M{
			"hash":  dv.Hash(),
			"value": dv.v.Nanoseconds(),
		},
	))
}

type DurationValueUnpackerBSON struct {
	H valuehash.Bytes `bson:"hash"`
	V int64           `bson:"value"`
}

func (dv *DurationValue) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uv DurationValueUnpackerBSON
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	dv.h = uv.H
	dv.v = time.Duration(uv.V)

	return nil
}
