package key

import (
	"bytes"

	"github.com/btcsuite/btcutil/base58"
	"golang.org/x/xerrors"
)

type Signature []byte

func (sg Signature) Bytes() []byte {
	return sg
}

func (sg Signature) String() string {
	return base58.Encode(sg)
}

func (sg Signature) IsValid([]byte) error {
	if len(sg) < 1 {
		return xerrors.Errorf("empty Signature")
	}

	return nil
}

func (sg Signature) Equal(ns Signature) bool {
	if len(sg) != len(ns) {
		return false
	}

	return bytes.Equal(sg, ns)
}
