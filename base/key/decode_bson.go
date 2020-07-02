package key

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
)

func (kd *KeyDecoder) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	if t != bsontype.String {
		return xerrors.Errorf("invalid marshaled type for KeyDecoder, %v", t)
	}

	s, _, ok := bsoncore.ReadString(b)
	if !ok {
		return xerrors.Errorf("can not read string")
	}

	if h, us, err := parseString(s); err != nil {
		return err
	} else {
		kd.h = h
		kd.s = us
	}

	return nil
}
