package key

import (
	"github.com/btcsuite/btcutil/base58"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func (sg Signature) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, sg.String()), nil
}

func (sg *Signature) UnmarshalBSONValue(_ bsontype.Type, b []byte) error {
	s, _, ok := bsoncore.ReadString(b)
	if !ok {
		return errors.Errorf("can not read string")
	}

	*sg = Signature(base58.Decode(s))

	return nil
}
