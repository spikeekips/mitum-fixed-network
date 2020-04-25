package state

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bv BytesValue) MarshalBSON() ([]byte, error) {
	return bson.Marshal(encoder.MergeBSONM(
		encoder.NewBSONHintedDoc(bv.Hint()),
		bson.M{
			"hash":  bv.h,
			"value": bv.v,
		},
	))
}

type BytesValueUnpackerBSON struct {
	H bson.Raw `bson:"hash"`
	V []byte   `bson:"value"`
}

func (bv *BytesValue) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
	var uv BytesValueUnpackerBSON
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	var err error
	var h valuehash.Hash
	if h, err = valuehash.Decode(enc, uv.H); err != nil {
		return err
	}

	bv.h = h
	bv.v = uv.V

	return nil
}
