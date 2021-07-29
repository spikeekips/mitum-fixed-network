package node

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/hint"
)

type BaseV0PackerBSON struct {
	HT hint.Hint     `bson:"_hint"`
	AD base.Address  `bson:"address"`
	PK key.Publickey `bson:"publickey"`
}

func (bn BaseV0) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(BaseV0PackerBSON{
		HT: bn.Hint(),
		AD: bn.address,
		PK: bn.publickey,
	})
}

type BaseV0UnpackerBSON struct {
	AD base.AddressDecoder  `bson:"address"`
	PK key.PublickeyDecoder `bson:"publickey"`
}

func (bn *BaseV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var nbn BaseV0UnpackerBSON
	if err := enc.Unmarshal(b, &nbn); err != nil {
		return err
	}

	return bn.unpack(enc, nbn.AD, nbn.PK)
}
