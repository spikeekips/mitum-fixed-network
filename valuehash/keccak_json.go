package valuehash

func (s256 SHA256) MarshalJSON() ([]byte, error) {
	return marshalJSON(s256)
}

func (s256 *SHA256) UnmarshalJSON(b []byte) error {
	h, err := unmarshalJSON(b)
	if err != nil {
		return err
	}

	if h, err := LoadSHA256FromString(h.Hash); err != nil {
		return err
	} else {
		*s256 = h.(SHA256)
	}

	return nil
}

func (s512 SHA512) MarshalJSON() ([]byte, error) {
	return marshalJSON(s512)
}

func (s512 *SHA512) UnmarshalJSON(b []byte) error {
	h, err := unmarshalJSON(b)
	if err != nil {
		return err
	}

	if h, err := LoadSHA512FromString(h.Hash); err != nil {
		return err
	} else {
		*s512 = h.(SHA512)
	}

	return nil
}
