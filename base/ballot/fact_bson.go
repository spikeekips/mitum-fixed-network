package ballot

import (
	"github.com/spikeekips/mitum/base"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
	"go.mongodb.org/mongo-driver/bson"
)

func (fact BaseFact) packerBSON() bson.M {
	return bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(fact.Hint()),
		bson.M{
			"hash":   fact.h,
			"height": fact.height,
			"round":  fact.round,
		})
}

type BaseFactUnpackerBSON struct {
	HI hint.Hint       `json:"_hint"`
	H  valuehash.Bytes `bson:"hash"`
	HT base.Height     `bson:"height"`
	R  base.Round      `bson:"round"`
}

func (fact *BaseFact) unpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uf BaseFactUnpackerBSON
	if err := enc.Unmarshal(b, &uf); err != nil {
		return err
	}

	return fact.unpack(enc, uf.HI, uf.H, uf.HT, uf.R)
}
