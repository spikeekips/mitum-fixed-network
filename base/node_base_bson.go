package base

import (
	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
)

type BaseNodeV0PackerBSON struct {
	HT hint.Hint     `bson:"_hint"`
	AD Address       `bson:"address"`
	PK key.Publickey `bson:"publickey"`
	UR string        `bson:"url"`
}

func (bn BaseNodeV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(BaseNodeV0PackerBSON{
		HT: bn.Hint(),
		AD: bn.address,
		PK: bn.publickey,
		UR: bn.url,
	})
}

type BaseNodeV0UnpackerBSON struct {
	AD AddressDecoder       `bson:"address"`
	PK key.PublickeyDecoder `bson:"publickey"`
	UR string               `bson:"url"`
}

func (bn *BaseNodeV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var nbn BaseNodeV0UnpackerBSON
	if err := enc.Unmarshal(b, &nbn); err != nil {
		return err
	}

	return bn.unpack(enc, nbn.AD, nbn.PK, nbn.UR)
}
