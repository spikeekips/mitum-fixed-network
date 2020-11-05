package key

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func marshalBSONStringKey(k Key) (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, k.String()), nil
}
