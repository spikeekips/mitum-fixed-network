package key // nolint

func (sp StellarPrivatekey) MarshalJSON() ([]byte, error) {
	return MarshalJSONKey(sp)
}

func (sp *StellarPrivatekey) UnmarshalJSON(b []byte) error {
	_, s, err := UnmarshalJSONKey(b)
	if err != nil {
		return err
	}

	if kp, err := NewStellarPrivatekeyFromString(s); err != nil {
		return err
	} else {
		sp.kp = kp.kp
	}

	return nil
}

func (sp StellarPublickey) MarshalJSON() ([]byte, error) {
	return MarshalJSONKey(sp)
}

func (sp *StellarPublickey) UnmarshalJSON(b []byte) error {
	_, s, err := UnmarshalJSONKey(b)
	if err != nil {
		return err
	}

	if kp, err := NewStellarPublickeyFromString(s); err != nil {
		return err
	} else {
		sp.kp = kp.kp
	}

	return nil
}
