package state

import (
	"reflect"

	"go.mongodb.org/mongo-driver/bson"

	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
)

func (nv NumberValue) MarshalBSON() ([]byte, error) {
	return bsonencoder.Marshal(bsonencoder.MergeBSONM(
		bsonencoder.NewHintedDoc(nv.Hint()),
		bson.M{
			"hash":  nv.Hash(),
			"value": nv.b,
			"type":  nv.t,
		},
	))
}

type NumberValueBSONUnpacker struct {
	H bson.Raw     `bson:"hash"`
	V []byte       `bson:"value"`
	T reflect.Kind `bson:"type"`
}

func (nv *NumberValue) UnpackBSON(b []byte, enc *bsonencoder.Encoder) error {
	var uv NumberValueBSONUnpacker
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	return nv.unpack(enc, uv.H, uv.V, uv.T)
}
