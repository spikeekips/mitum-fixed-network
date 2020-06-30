package state

import (
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (bv BytesValue) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(bv.Hint()),
		bson.M{
			"hash":  bv.h,
			"value": bv.v,
		},
	))
}

type BytesValueUnpackerBSON struct {
	H valuehash.Bytes `bson:"hash"`
	V []byte          `bson:"value"`
}

func (bv *BytesValue) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uv BytesValueUnpackerBSON
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	if uv.H.Empty() {
		return xerrors.Errorf("empty hash found")
	}

	bv.h = uv.H
	bv.v = uv.V

	return nil
}
