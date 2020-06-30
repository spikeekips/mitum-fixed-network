package state

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (sv SliceValue) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(sv.Hint()),
		bson.M{
			"hash":  sv.Hash(),
			"value": sv.v,
		},
	))
}

func (sv *SliceValue) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uv struct {
		H valuehash.Bytes `bson:"hash"`
		V []bson.Raw      `bson:"value"`
	}

	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	bValue := make([][]byte, len(uv.V))
	for i, v := range uv.V {
		bValue[i] = v
	}

	return sv.unpack(enc, uv.H, bValue)
}
