package key

func (bt BTCPrivatekey) MarshalBSON() ([]byte, error) {
	return MarshalBSONKey(bt)
}

func (bt *BTCPrivatekey) UnmarshalBSON(b []byte) error {
	var key string
	if _, s, err := UnmarshalBSONKey(b); err != nil {
		return err
	} else {
		key = s
	}

	return bt.unpack(key)
}

func (bt BTCPublickey) MarshalBSON() ([]byte, error) {
	return MarshalBSONKey(bt)
}

func (bt *BTCPublickey) UnmarshalBSON(b []byte) error {
	var key string
	if _, s, err := UnmarshalBSONKey(b); err != nil {
		return err
	} else {
		key = s
	}

	return bt.unpack(key)
}
