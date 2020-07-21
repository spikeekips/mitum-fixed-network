package key

import "go.mongodb.org/mongo-driver/bson/bsontype"

func (sp StellarPrivatekey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(sp)
}

func (sp StellarPublickey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(sp)
}
