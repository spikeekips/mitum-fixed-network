package operation

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (bo BaseOperation) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(bo.Hint()),
		bson.M{
			"hash":       bo.h,
			"fact":       bo.fact,
			"fact_signs": bo.fs,
		},
	))
}

type BaseOperationBSONUnpacker struct {
	HT hint.Hint       `bson:"_hint"`
	H  valuehash.Bytes `bson:"hash"`
	FC bson.Raw        `bson:"fact"`
	FS []bson.Raw      `bson:"fact_signs"`
}

func (bo *BaseOperation) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ubo BaseOperationBSONUnpacker
	if err := enc.Unmarshal(b, &ubo); err != nil {
		return err
	}

	fs := make([][]byte, len(ubo.FS))
	for i := range ubo.FS {
		fs[i] = ubo.FS[i]
	}

	return bo.unpack(enc, ubo.HT, ubo.H, ubo.FC, fs)
}
