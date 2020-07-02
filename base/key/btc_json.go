package key

func (bt BTCPrivatekey) MarshalJSON() ([]byte, error) {
	return marshalJSONStringKey(bt)
}

func (bt *BTCPrivatekey) UnmarshalJSON(b []byte) error {
	if k, err := NewBTCPrivatekeyFromString(string(b)); err != nil {
		return err
	} else {
		*bt = k
	}

	return nil
}

func (bt BTCPublickey) MarshalJSON() ([]byte, error) {
	return marshalJSONStringKey(bt)
}

func (bt *BTCPublickey) UnmarshalJSON(b []byte) error {
	if k, err := NewBTCPublickeyFromString(string(b)); err != nil {
		return err
	} else {
		*bt = k
	}

	return nil
}
