package valuehash

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

func (hs Bytes) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONValue(hs)
}

func (hs *Bytes) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	if t != bsontype.String {
		if t == bsontype.Null {
			*hs = NewBytes(nil)

			return nil
		}

		return errors.Errorf("invalid marshaled type for Hash, %v", t)
	}

	bt, err := unmarshalBSONValue(b)
	if err != nil {
		return err
	}
	*hs = bt

	return nil
}
