package node

import (
	"github.com/spikeekips/mitum/hash"
)

var (
	AddressHashHint string = "na"
)

type Address struct {
	hash.Hash
}

func NewAddress(b []byte) (Address, error) {
	h, err := hash.NewDoubleSHAHash(AddressHashHint, b)
	if err != nil {
		return Address{}, err
	}

	return Address{Hash: h}, nil
}

func (ad Address) Equal(nad Address) bool {
	return ad.Hash.Equal(nad.Hash)
}

func IsAddress(h Address) bool {
	if h.Hint() != AddressHashHint {
		return false
	}

	return true
}
