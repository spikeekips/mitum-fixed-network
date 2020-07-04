package localtime

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
)

func (jt JSONTime) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, RFC3339(jt.Time)), nil
}

func (jt *JSONTime) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	if t != bsontype.String {
		return xerrors.Errorf("invalid marshaled type for JSONTime, %v", t)
	}

	if s, _, ok := bsoncore.ReadString(b); !ok {
		return xerrors.Errorf("can not read string")
	} else if t, err := ParseTimeFromRFC3339(s); err != nil {
		return err
	} else {
		jt.Time = t
	}

	return nil
}
