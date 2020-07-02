package key

import (
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

func (ep EtherPrivatekey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(ep)
}

func (ep *EtherPrivatekey) UnmarshalBSON(b []byte) error {
	if k, err := NewEtherPrivatekeyFromString(string(b)); err != nil {
		return err
	} else {
		*ep = k
	}

	return nil
}

func (ep EtherPublickey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(ep)
}

func (ep *EtherPublickey) UnmarshalBSON(b []byte) error {
	if k, err := NewEtherPublickeyFromString(string(b)); err != nil {
		return err
	} else {
		*ep = k
	}

	return nil
}
