package state

import (
	"reflect"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (nv NumberValue) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(nv.Hint()),
		bson.M{
			"hash":  nv.Hash(),
			"value": nv.b,
			"type":  nv.t,
		},
	))
}

type NumberValueBSONUnpacker struct {
	H valuehash.Bytes `bson:"hash"`
	V []byte          `bson:"value"`
	T reflect.Kind    `bson:"type"`
}

func (nv *NumberValue) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uv NumberValueBSONUnpacker
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	return nv.unpack(enc, uv.H, uv.V, uv.T)
}
