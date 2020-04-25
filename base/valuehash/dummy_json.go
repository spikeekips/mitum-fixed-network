package valuehash

func (dm Dummy) MarshalJSON() ([]byte, error) {
	return marshalJSON(dm)
}

func (dm *Dummy) UnmarshalJSON(b []byte) error {
	h, err := unmarshalJSON(b)
	if err != nil {
		return err
	}

	return dm.unpack(h.Hash)
}
