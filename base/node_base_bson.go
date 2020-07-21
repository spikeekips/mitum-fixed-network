package base

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
)

type BaseNodeV0PackerBSON struct {
	HT hint.Hint     `bson:"_hint"`
	AD Address       `bson:"address"`
	PK key.Publickey `bson:"publickey"`
}

func (bn BaseNodeV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(BaseNodeV0PackerBSON{
		HT: bn.Hint(),
		AD: bn.address,
		PK: bn.publickey,
	})
}

type BaseNodeV0UnpackerBSON struct {
	AD bson.Raw             `bson:"address"`
	PK key.PublickeyDecoder `bson:"publickey"`
}

func (bn *BaseNodeV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var nbn BaseNodeV0UnpackerBSON
	if err := enc.Unmarshal(b, &nbn); err != nil {
		return err
	}

	return bn.unpack(enc, nbn.AD, nbn.PK)
}
