package key // nolint

func (sp *StellarPrivatekey) unpack(s string) error {
	if kp, err := NewStellarPrivatekeyFromString(s); err != nil {
		return err
	} else {
		sp.kp = kp.kp
	}

	return nil
}

func (sp *StellarPublickey) unpack(s string) error {
	if kp, err := NewStellarPublickeyFromString(s); err != nil {
		return err
	} else {
		sp.kp = kp.kp
	}

	return nil
}
