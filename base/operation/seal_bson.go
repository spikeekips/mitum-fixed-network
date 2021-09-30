package operation

import (
	"github.com/spikeekips/mitum/base/seal"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson"
)

func (sl BaseSeal) MarshalBSON() ([]byte, error) {
	m := sl.BaseSeal.BSONPacker()
	m["operations"] = sl.ops

	return bsonenc.Marshal(m)
}

type BaseSealBSONUnpack struct {
	OPS bson.Raw `bson:"operations"`
}

func (sl *BaseSeal) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ub seal.BaseSeal
	if err := ub.UnpackBSON(b, enc); err != nil {
		return err
	}

	var usl BaseSealBSONUnpack
	if err := enc.Unmarshal(b, &usl); err != nil {
		return err
	}

	return sl.unpack(enc, ub, usl.OPS)
}
