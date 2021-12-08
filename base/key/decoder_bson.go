package key

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func (kd *KeyDecoder) UnmarshalBSON(b []byte) error {
	if len(b) < 1 {
		return nil
	}

	s, _, ok := bsoncore.ReadString(b)
	if !ok {
		return util.NotFoundError.Errorf("not string type in bson")
	}

	p, ty, err := hint.ParseFixedTypedString(s, KeyTypeSize)
	if err != nil {
		return err
	}

	kd.ty = ty
	kd.b = []byte(p)

	return nil
}
