package key

func (ep EtherPrivatekey) MarshalJSON() ([]byte, error) {
	return marshalJSONStringKey(ep)
}

func (ep *EtherPrivatekey) UnmarshalJSON(b []byte) error {
	if k, err := NewEtherPrivatekeyFromString(string(b)); err != nil {
		return err
	} else {
		*ep = k
	}

	return nil
}

func (ep EtherPublickey) MarshalJSON() ([]byte, error) {
	return marshalJSONStringKey(ep)
}

func (ep *EtherPublickey) UnmarshalJSON(b []byte) error {
	if k, err := NewEtherPublickeyFromString(string(b)); err != nil {
		return err
	} else {
		*ep = k
	}

	return nil
}
