package seal

import (
	"time"

	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (sl BaseSeal) BSONPacker() map[string]interface{} {
	return bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(sl.Hint()),
		bson.M{
			"hash":      sl.h,
			"body_hash": sl.bodyHash,
			"signer":    sl.signer,
			"signature": sl.signature,
			"signed_at": sl.signedAt,
		},
	)
}

type BaseSealBSONUnpack struct {
	HH valuehash.Bytes      `bson:"hash"`
	BH valuehash.Bytes      `bson:"body_hash"`
	SN key.PublickeyDecoder `bson:"signer"`
	SG key.Signature        `bson:"signature"`
	SA time.Time            `bson:"signed_at"`
}

func (sl *BaseSeal) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var usl BaseSealBSONUnpack
	if err := enc.Unmarshal(b, &usl); err != nil {
		return err
	}

	return sl.unpack(enc, usl.HH, usl.BH, usl.SN, usl.SG, usl.SA)
}
