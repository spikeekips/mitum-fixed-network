package valuehash

func (s256 *SHA256) unpack(s string) error {
	if h, err := LoadSHA256FromString(s); err != nil {
		return err
	} else {
		*s256 = h.(SHA256)
	}

	return nil
}

func (s512 *SHA512) unpack(s string) error {
	if h, err := LoadSHA512FromString(s); err != nil {
		return err
	} else {
		*s512 = h.(SHA512)
	}

	return nil
}
