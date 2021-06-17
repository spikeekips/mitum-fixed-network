package hint

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
	"golang.org/x/xerrors"
)

func (hs HintedString) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, hs.String()), nil
}

func (hs *HintedString) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	if len(b) < 1 {
		return nil
	}

	switch t {
	case bsontype.Null:
		return nil
	case bsontype.String:
	default:
		return xerrors.Errorf("invalid marshaled type for HintedString, %v", t)
	}

	i, _, ok := bsoncore.ReadString(b)
	if !ok {
		return xerrors.Errorf("can not read string")
	}

	return hs.UnmarshalText([]byte(i))
}
