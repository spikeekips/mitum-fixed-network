package operation

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	"go.mongodb.org/mongo-driver/bson"
)

func (sl Seal) MarshalBSON() ([]byte, error) {
	return bson.Marshal(encoder.MergeBSONM(
		encoder.NewBSONHintedDoc(sl.Hint()),
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
	H   bson.Raw      `bson:"hash"`
	BH  bson.Raw      `bson:"body_hash"`
	SN  bson.Raw      `bson:"signer"`
	SG  key.Signature `bson:"signature"`
	SA  time.Time     `bson:"signed_at"`
	OPS []bson.Raw    `bson:"operations"`
}

func (sl *Seal) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
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
