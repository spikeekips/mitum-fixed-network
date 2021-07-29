package util

import "strings"

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

func ParseBoolInQuery(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "t", "true", "1", "y", "yes":
		return true
	default:
		return false
	}
}
