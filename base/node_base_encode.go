package base

import (
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bn *BaseNodeV0) unpack(enc encoder.Encoder, bad, bpk []byte) error {
	var address Address
	if a, err := DecodeAddress(enc, bad); err != nil {
		return err
	} else {
		address = a
	}

	var pk key.Publickey
	if p, err := key.DecodePublickey(enc, bpk); err != nil {
		return err
	} else {
		pk = p
	}

	bn.address = address
	bn.publickey = pk

	return nil
}
