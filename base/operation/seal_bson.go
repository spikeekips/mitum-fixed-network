package operation

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (sl BaseSeal) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(sl.Hint()),
		bson.M{
			"hash":       sl.h,
			"body_hash":  sl.bodyHash,
			"signer":     sl.signer,
			"signature":  sl.signature,
			"signed_at":  sl.signedAt,
			"operations": sl.ops,
		},
	))
}

type SealBSONUnpack struct {
	H   valuehash.Bytes      `bson:"hash"`
	BH  valuehash.Bytes      `bson:"body_hash"`
	SN  key.PublickeyDecoder `bson:"signer"`
	SG  key.Signature        `bson:"signature"`
	SA  time.Time            `bson:"signed_at"`
	OPS []bson.Raw           `bson:"operations"`
}

func (sl *BaseSeal) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var usl SealBSONUnpack
	if err := enc.Unmarshal(b, &usl); err != nil {
		return err
	}

	ops := make([][]byte, len(usl.OPS))
	for i, b := range usl.OPS {
		ops[i] = b
	}

	return sl.unpack(enc, usl.H, usl.BH, usl.SN, usl.SG, usl.SA, ops)
}
