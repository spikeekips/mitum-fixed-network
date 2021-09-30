package network

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (sl HandoverSealV0) MarshalBSON() ([]byte, error) {
	m := sl.BaseSeal.BSONPacker()
	m["address"] = sl.ad
	m["conninfo"] = sl.ci

	return bsonenc.Marshal(m)
}

type HandoverSealV0BSONUnpack struct {
	AD base.AddressDecoder `bson:"address"`
	CI bson.Raw            `bson:"conninfo"`
}

func (sl *HandoverSealV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ub seal.BaseSeal
	if err := ub.UnpackBSON(b, enc); err != nil {
		return err
	}

	var usl HandoverSealV0BSONUnpack
	if err := enc.Unmarshal(b, &usl); err != nil {
		return err
	}

	return sl.unpack(enc, ub, usl.AD, usl.CI)
}
