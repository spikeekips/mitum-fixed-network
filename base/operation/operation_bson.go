package operation

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (bo BaseOperation) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bo.BSONM())
}

func (bo BaseOperation) BSONM() bson.M {
	return bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(bo.Hint()),
		bson.M{
			"hash":       bo.h,
			"fact":       bo.fact,
			"fact_signs": bo.fs,
		},
	)
}

type BaseOperationBSONUnpacker struct {
	H  valuehash.Bytes `bson:"hash"`
	FC bson.Raw        `bson:"fact"`
	FS bson.Raw        `bson:"fact_signs"`
}

func (bo *BaseOperation) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo BaseOperationBSONUnpacker
	if err := enc.Unmarshal(b, &ubo); err != nil {
		return err
	}

	return bo.unpack(enc, ubo.H, ubo.FC, ubo.FS)
}
