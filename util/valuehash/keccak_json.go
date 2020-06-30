package valuehash

func (hs SHA256) MarshalJSON() ([]byte, error) {
	return marshalJSON(hs)
}

func (hs SHA512) MarshalJSON() ([]byte, error) {
	return marshalJSON(hs)
}
