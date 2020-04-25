package key

func (ep EtherPrivatekey) MarshalJSON() ([]byte, error) {
	return MarshalJSONKey(ep)
}

func (ep *EtherPrivatekey) UnmarshalJSON(b []byte) error {
	var key string
	if h, s, err := UnmarshalJSONKey(b); err != nil {
		return err
	} else if err := ep.Hint().IsCompatible(h); err != nil {
		return err
	} else {
		key = s
	}

	kp, err := NewEtherPrivatekeyFromString(key)
	if err != nil {
		return err
	}

	ep.pk = kp.pk

	return nil
}

func (ep EtherPublickey) MarshalJSON() ([]byte, error) {
	return MarshalJSONKey(ep)
}

func (ep *EtherPublickey) UnmarshalJSON(b []byte) error {
	var key string
	if h, s, err := UnmarshalJSONKey(b); err != nil {
		return err
	} else if err := ep.Hint().IsCompatible(h); err != nil {
		return err
	} else {
		key = s
	}

	kp, err := NewEtherPublickey(key)
	if err != nil {
		return err
	}

	ep.pk = kp.pk

	return nil
}
