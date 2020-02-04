package key // nolint

func (bt BTCPrivatekey) MarshalJSON() ([]byte, error) {
	return MarshalJSONKey(bt)
}

func (bt *BTCPrivatekey) UnmarshalJSON(b []byte) error {
	var key string
	if h, s, err := UnmarshalJSONKey(b); err != nil {
		return err
	} else if err := bt.Hint().IsCompatible(h); err != nil {
		return err
	} else {
		key = s
	}

	kp, err := NewBTCPrivatekeyFromString(key)
	if err != nil {
		return err
	}

	bt.wif = kp.wif

	return nil
}

func (bt BTCPublickey) MarshalJSON() ([]byte, error) {
	return MarshalJSONKey(bt)
}

func (bt *BTCPublickey) UnmarshalJSON(b []byte) error {
	var key string
	if h, s, err := UnmarshalJSONKey(b); err != nil {
		return err
	} else if err := bt.Hint().IsCompatible(h); err != nil {
		return err
	} else {
		key = s
	}

	kp, err := NewBTCPublickeyFromString(key)
	if err != nil {
		return err
	}

	bt.pk = kp.pk

	return nil
}
