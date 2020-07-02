package key

import "go.mongodb.org/mongo-driver/bson/bsontype"

func (sp StellarPrivatekey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(sp)
}

func (sp *StellarPrivatekey) UnmarshalBSON(b []byte) error {
	if k, err := NewStellarPrivatekeyFromString(string(b)); err != nil {
		return err
	} else {
		*sp = k
	}

	return nil
}

func (sp StellarPublickey) MarshalBSONValue() (bsontype.Type, []byte, error) {
	return marshalBSONStringKey(sp)
}

func (sp *StellarPublickey) UnmarshalBSON(b []byte) error {
	if k, err := NewStellarPublickeyFromString(string(b)); err != nil {
		return err
	} else {
		*sp = k
	}

	return nil
}
