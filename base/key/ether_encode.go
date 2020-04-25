package key

func (ep *EtherPrivatekey) unpack(s string) error {
	kp, err := NewEtherPrivatekeyFromString(s)
	if err != nil {
		return err
	}

	ep.pk = kp.pk

	return nil
}

func (ep *EtherPublickey) unpack(s string) error {
	kp, err := NewEtherPublickey(s)
	if err != nil {
		return err
	}

	ep.pk = kp.pk

	return nil
}
