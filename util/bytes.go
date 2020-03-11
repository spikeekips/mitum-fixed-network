package util

import "bytes"

type Byter interface {
	Bytes() []byte
}

func NewByter(b []byte) Byter {
	return bytes.NewBuffer(b)
}

func CopyBytes(b []byte) []byte {
	n := make([]byte, len(b))
	copy(n, b)

	return b
}
