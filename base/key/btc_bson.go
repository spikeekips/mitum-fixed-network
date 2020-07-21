package key

import "go.mongodb.org/mongo-driver/bson/bsontype"

func (bt BTCPrivatekey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(bt)
}

func (bt BTCPublickey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(bt)
}
