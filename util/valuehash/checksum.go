package valuehash

import (
	"crypto/sha256"
	"fmt"
)

func SHA256Checksum(b []byte) string {
	sha := sha256.New()
	_, _ = sha.Write(b)

	return fmt.Sprintf("%x", sha.Sum(nil))
}
