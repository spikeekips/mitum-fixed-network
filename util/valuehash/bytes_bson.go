package valuehash

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"golang.org/x/xerrors"
)

func (hs Bytes) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONValue(hs)
}

func (hs *Bytes) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	if t != bsontype.String {
		return xerrors.Errorf("invalid marshaled type for Hash, %v", t)
	}

	if bt, err := unmarshalBSONValue(b); err != nil {
		return err
	} else {
		*hs = bt
	}

	return nil
}
