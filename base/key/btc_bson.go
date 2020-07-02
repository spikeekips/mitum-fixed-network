package key

import "go.mongodb.org/mongo-driver/bson/bsontype"

func (bt BTCPrivatekey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(bt)
}

func (bt *BTCPrivatekey) UnmarshalBSON(b []byte) error {
	if k, err := NewBTCPrivatekeyFromString(string(b)); err != nil {
		return err
	} else {
		*bt = k
	}

	return nil
}

func (bt BTCPublickey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(bt)
}

func (bt *BTCPublickey) UnmarshalBSON(b []byte) error {
	if k, err := NewBTCPublickeyFromString(string(b)); err != nil {
		return err
	} else {
		*bt = k
	}

	return nil
}
