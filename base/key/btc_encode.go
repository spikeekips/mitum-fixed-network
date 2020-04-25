package key

func (bt *BTCPrivatekey) unpack(s string) error {
	kp, err := NewBTCPrivatekeyFromString(s)
	if err != nil {
		return err
	}

	bt.wif = kp.wif

	return nil
}

func (bt *BTCPublickey) unpack(s string) error {
	kp, err := NewBTCPublickeyFromString(s)
	if err != nil {
		return err
	}

	bt.pk = kp.pk

	return nil
}
