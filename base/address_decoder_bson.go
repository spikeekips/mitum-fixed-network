package base

import (
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func (ad *AddressDecoder) UnmarshalBSON(b []byte) error {
	if len(b) < 1 {
		return nil
	}

	s, _, ok := bsoncore.ReadString(b)
	if !ok {
		return util.NotFoundError.Errorf("not string type in bson")
	}

	p, ty, err := hint.ParseFixedTypedString(s, AddressTypeSize)
	if err != nil {
		return err
	}

	ad.ty = ty
	ad.b = []byte(p)

	return nil
}
