package state

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/util/encoder"
)

func (hv HintedValue) MarshalBSON() ([]byte, error) {
	return bson.Marshal(encoder.MergeBSONM(
		encoder.NewBSONHintedDoc(hv.Hint()),
		bson.M{
			"hash":  hv.Hash(),
			"value": hv.v,
		},
	))
}

type HintedValueUnpackerBSON struct {
	H bson.Raw `bson:"hash"`
	V bson.Raw `bson:"value"`
}

func (hv *HintedValue) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
	var uv HintedValueUnpackerBSON
	if err := enc.Unmarshal(b, &uv); err != nil {
		return err
	}

	decoded, err := enc.DecodeByHint(uv.V)
	if err != nil {
		return err
	}

	hv.v = decoded

	return nil
}
