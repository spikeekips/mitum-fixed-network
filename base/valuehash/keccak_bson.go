package valuehash

func (s256 SHA256) MarshalBSON() ([]byte, error) {
	return marshalBSON(s256)
}

func (s256 *SHA256) UnmarshalBSON(b []byte) error {
	h, err := unmarshalBSON(b)
	if err != nil {
		return err
	}

	return s256.unpack(h.Hash)
}

func (s512 SHA512) MarshalBSON() ([]byte, error) {
	return marshalBSON(s512)
}

func (s512 *SHA512) UnmarshalBSON(b []byte) error {
	h, err := unmarshalBSON(b)
	if err != nil {
		return err
	}

	return s512.unpack(h.Hash)
}
