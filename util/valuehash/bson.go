package valuehash

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
)

func marshalBSONValue(h Hash) (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, h.String()), nil
}

func unmarshalBSONValue(b []byte) (Bytes, error) {
	s, ok := (bson.RawValue{Type: bsontype.String, Value: b}).StringValueOK()
	if !ok {
		return Bytes{}, xerrors.Errorf("invalid encoded input for Hash")
	}

	return NewBytes(fromString(s)), nil
}
