package base

import (
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bn *BaseNodeV0) unpack(enc encoder.Encoder, bad AddressDecoder, bpk key.PublickeyDecoder, url string) error {
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
	bn.url = url

	return nil
}
