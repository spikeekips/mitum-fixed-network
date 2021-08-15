package state

import (
	"github.com/pkg/errors"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (sv StringValue) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(sv.Hint()),
		bson.M{
			"hash":  sv.Hash(),
			"value": sv.v,
		},
	))
}

type StringValueUnpackerBSON struct {
	H valuehash.Bytes `bson:"hash"`
	V string          `bson:"value"`
}

func (sv *StringValue) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uv StringValueUnpackerBSON
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	if uv.H.IsEmpty() {
		return errors.Errorf("empty previous_block hash found")
	}

	sv.h = uv.H
	sv.v = uv.V

	return nil
}
