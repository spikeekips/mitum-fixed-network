package key

import (
	"bytes"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/mitum/util/isvalid"
)

type Signature []byte

func NewSignatureFromString(s string) Signature {
	return Signature(base58.Decode(s))
}

func (sg Signature) Bytes() []byte {
	return sg
}

func (sg Signature) String() string {
	return base58.Encode(sg)
}

func (sg Signature) IsValid([]byte) error {
	if len(sg) < 1 {
		return isvalid.InvalidError.Errorf("empty Signature")
	}

	return nil
}

func (sg Signature) Equal(ns Signature) bool {
	if len(sg) != len(ns) {
		return false
	}

	return bytes.Equal(sg, ns)
}
