package localtime

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
)

func (t Time) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, t.Normalize().RFC3339()), nil
}

func (t *Time) UnmarshalBSONValue(ty bsontype.Type, b []byte) error {
	if ty != bsontype.String {
		return xerrors.Errorf("invalid marshaled type for localtime.Time, %v", ty)
	}

	if s, _, ok := bsoncore.ReadString(b); !ok {
		return xerrors.Errorf("can not read string for localtime.Time")
	} else if err := t.UnmarshalText([]byte(s)); err != nil {
		return err
	} else {
		return nil
	}
}
