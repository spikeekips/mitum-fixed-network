package state

import (
	"github.com/spikeekips/mitum/util/encoder"
	"go.mongodb.org/mongo-driver/bson"
)

func (sv SliceValue) MarshalBSON() ([]byte, error) {
	return bson.Marshal(encoder.MergeBSONM(
		encoder.NewBSONHintedDoc(sv.Hint()),
		bson.M{
			"hash":  sv.Hash(),
			"value": sv.v,
		},
	))
}

func (sv *SliceValue) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
	var uv struct {
		H bson.Raw   `bson:"hash"`
		V []bson.Raw `bson:"value"`
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
