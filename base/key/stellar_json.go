package key

func (sp StellarPrivatekey) MarshalJSON() ([]byte, error) {
	return marshalJSONStringKey(sp)
}

func (sp *StellarPrivatekey) UnmarshalJSON(b []byte) error {
	if k, err := NewStellarPrivatekeyFromString(string(b)); err != nil {
		return err
	} else {
		*sp = k
	}

	return nil
}

func (sp StellarPublickey) MarshalJSON() ([]byte, error) {
	return marshalJSONStringKey(sp)
}

func (sp *StellarPublickey) UnmarshalJSON(b []byte) error {
	if k, err := NewStellarPublickeyFromString(string(b)); err != nil {
		return err
	} else {
		*sp = k
	}

	return nil
}
