package localtime

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func (t Time) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, t.Normalize().RFC3339()), nil
}

func (t *Time) UnmarshalBSONValue(ty bsontype.Type, b []byte) error {
	if ty != bsontype.String {
		return errors.Errorf("invalid marshaled type for localtime.Time, %v", ty)
	}

	if s, _, ok := bsoncore.ReadString(b); !ok {
		return errors.Errorf("can not read string for localtime.Time")
	} else if err := t.UnmarshalText([]byte(s)); err != nil {
		return err
	} else {
		return nil
	}
}
