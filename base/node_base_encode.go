package base

import (
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bn *BaseNodeV0) unpack(enc encoder.Encoder, bad AddressDecoder, bpk key.PublickeyDecoder) error {
	var address Address
	if a, err := bad.Encode(enc); err != nil {
		return err
	} else {
		address = a
	}

	var pk key.Publickey
	if k, err := bpk.Encode(enc); err != nil {
		return err
	} else {
		pk = k
	}

	bn.address = address
	bn.publickey = pk

	return nil
}
