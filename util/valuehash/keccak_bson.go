package valuehash

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

func (hs SHA256) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONValue(hs)
}

func (hs SHA512) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONValue(hs)
}
