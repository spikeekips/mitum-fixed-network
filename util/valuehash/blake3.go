package valuehash

import (
	"github.com/zeebo/blake3"
)

func NewBlake3256(b []byte) L32 {
	return L32(blake3.Sum256(b))
}
