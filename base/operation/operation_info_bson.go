package operation

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (oi OperationInfoV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(oi.Hint()),
		bson.M{
			"operation": oi.oh,
			"seal":      oi.sh,
		},
	))
}

type OperationInfoV0UnpackerBSON struct {
	OH valuehash.Bytes `bson:"operation"`
	SH valuehash.Bytes `bson:"seal"`
}

func (oi *OperationInfoV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uoi OperationInfoV0UnpackerBSON
	if err := enc.Unmarshal(b, &uoi); err != nil {
		return err
	}

	return oi.unpack(enc, uoi.OH, uoi.SH)
}
