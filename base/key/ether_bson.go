package key

func (ep EtherPrivatekey) MarshalBSON() ([]byte, error) {
	return MarshalBSONKey(ep)
}

func (ep *EtherPrivatekey) UnmarshalBSON(b []byte) error {
	var key string
	if _, s, err := UnmarshalBSONKey(b); err != nil {
		return err
	} else {
		key = s
	}

	return ep.unpack(key)
}

func (ep EtherPublickey) MarshalBSON() ([]byte, error) {
	return MarshalBSONKey(ep)
}

func (ep *EtherPublickey) UnmarshalBSON(b []byte) error {
	var key string
	if _, s, err := UnmarshalBSONKey(b); err != nil {
		return err
	} else {
		key = s
	}

	return ep.unpack(key)
}
