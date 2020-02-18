package util

func CopyBytes(b []byte) []byte {
	n := make([]byte, len(b))
	copy(n, b)

	return b
}
