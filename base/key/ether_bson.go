package key

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

func (ep EtherPrivatekey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(ep)
}

func (ep EtherPublickey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(ep)
}
