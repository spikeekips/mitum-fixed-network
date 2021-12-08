package key

import (
	"github.com/pkg/errors"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

func unmarshalbson(t bsontype.Type, b []byte, parser func(string) (Key, error)) (Key, error) {
	if t != bsontype.String {
		return nil, errors.Errorf("invalid marshaled type for privatekey, %v", t)
	}

	s, _, ok := bsoncore.ReadString(b)
	if !ok {
		return nil, errors.Errorf("can not read string")
	}

	return parser(s)
}

func (k BasePrivatekey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, k.String()), nil
}

func (k *BasePrivatekey) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	i, err := unmarshalbson(t, b, func(s string) (Key, error) {
		return ParseBasePrivatekey(s)
	})
	if err != nil {
		return err
	}

	uk, ok := i.(BasePrivatekey)
	if !ok {
		return errors.Errorf("not privatekey: %T", uk)
	}

	*k = uk

	return nil
}

func (k *BasePrivatekey) UnpackBSON(b []byte, _ *bsonenc.Encoder) error {
	uk, err := LoadBasePrivatekey(string(b))
	if err != nil {
		return err
	}

	*k = uk

	return nil
}

func (k BasePublickey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return bsontype.String, bsoncore.AppendString(nil, k.String()), nil
}

func (k *BasePublickey) UnmarshalBSONValue(t bsontype.Type, b []byte) error {
	i, err := unmarshalbson(t, b, func(s string) (Key, error) {
		return ParseBasePublickey(s)
	})
	if err != nil {
		return err
	}

	uk, ok := i.(BasePublickey)
	if !ok {
		return errors.Errorf("not publickey: %T", uk)
	}

	*k = uk

	return nil
}

func (k *BasePublickey) UnpackBSON(b []byte, _ *bsonenc.Encoder) error {
	uk, err := LoadBasePublickey(string(b))
	if err != nil {
		return err
	}

	*k = uk

	return nil
}
