package state

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (hv HintedValue) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(hv.Hint()),
		bson.M{
			"hash":  hv.Hash(),
			"value": hv.v,
		},
	))
}

type HintedValueUnpackerBSON struct {
	V bson.Raw `bson:"value"`
}

func (hv *HintedValue) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uv HintedValueUnpackerBSON
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	decoded, err := enc.Decode(uv.V)
	if err != nil {
		return err
	}

	hv.v = decoded

	return nil
}
