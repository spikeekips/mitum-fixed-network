package keypair

import "github.com/btcsuite/btcutil/base58"

type Signature []byte

func (s Signature) String() string {
	return base58.Encode(s)
}

func (s Signature) Equal(ns Signature) bool {
	if len(s) != len(ns) {
		return false
	}

	for i, b := range s {
		if b != ns[i] {
			return false
		}
	}

	return true
}
