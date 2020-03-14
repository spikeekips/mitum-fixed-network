package util

func BoolToBytes(b bool) []byte {
	var i int
	if b {
		i = 1
	}
	return IntToBytes(i)
}

func BytesToBool(b []byte) (bool, error) {
	i, err := BytesToInt(b)
	if err != nil {
		return false, err
	}

	return i != 0, nil
}
