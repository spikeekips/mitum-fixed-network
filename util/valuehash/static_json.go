package valuehash

func (h L32) MarshalJSON() ([]byte, error) {
	return marshalJSON(h)
}

func (h L64) MarshalJSON() ([]byte, error) {
	return marshalJSON(h)
}
