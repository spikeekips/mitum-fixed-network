package valuehash

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

func (h L32) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONValue(h)
}

func (h L64) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONValue(h)
}
