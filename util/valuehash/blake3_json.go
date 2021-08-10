package valuehash

func (hs Blake3256) MarshalJSON() ([]byte, error) {
	return marshalJSON(hs)
}
