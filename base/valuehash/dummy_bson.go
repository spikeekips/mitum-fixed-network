package valuehash

func (dm Dummy) MarshalBSON() ([]byte, error) {
	return marshalBSON(dm)
}

func (dm *Dummy) UnmarshalBSON(b []byte) error {
	h, err := unmarshalBSON(b)
	if err != nil {
		return err
	}

	return dm.unpack(h.Hash)
}
