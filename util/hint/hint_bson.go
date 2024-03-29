package hint

import (
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func (ht Hint) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, ht.String()), nil
}

func (ht *Hint) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	if t != bsontype.String {
		return errors.Errorf("invalid marshaled type for hint, %v", t)
	}

	if i, _, ok := bsoncore.ReadString(b); !ok {
		return errors.Errorf("can not read string")
	} else if j, err := ParseHint(i); err != nil {
		return err
	} else {
		*ht = j

		return nil
	}
}
