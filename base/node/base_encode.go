package node

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bn *BaseV0) unpack(enc encoder.Encoder, bad base.AddressDecoder, bpk key.PublickeyDecoder) error {
	address, err := bad.Encode(enc)
	if err != nil {
		return err
	}

	pk, err := bpk.Encode(enc)
	if err != nil {
		return err
	}

	bn.address = address
	bn.publickey = pk

	return nil
}
