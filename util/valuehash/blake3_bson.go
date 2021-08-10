package valuehash

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

func (hs Blake3256) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONValue(hs)
}
