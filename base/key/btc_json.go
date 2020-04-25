package key // nolint

func (bt BTCPrivatekey) MarshalJSON() ([]byte, error) {
	return MarshalJSONKey(bt)
}

func (bt *BTCPrivatekey) UnmarshalJSON(b []byte) error {
	var key string
	if _, s, err := UnmarshalJSONKey(b); err != nil {
		return err
	} else {
		key = s
	}

	return bt.unpack(key)
}

func (bt BTCPublickey) MarshalJSON() ([]byte, error) {
	return MarshalJSONKey(bt)
}

func (bt *BTCPublickey) UnmarshalJSON(b []byte) error {
	var key string
	if _, s, err := UnmarshalJSONKey(b); err != nil {
		return err
	} else {
		key = s
	}

	return bt.unpack(key)
}
