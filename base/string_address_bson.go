package base

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func (ad StringAddress) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, ad.String()), nil
}

func (ad *StringAddress) UnpackBSON(b []byte, _ *bsonenc.Encoder) error {
	*ad = NewStringAddress(string(b))

	return nil
}
