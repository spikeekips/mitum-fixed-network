package network

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/encoder"
)

func (sl *HandoverSealV0) unpack(enc encoder.Encoder, ub seal.BaseSeal, bad base.AddressDecoder, bci []byte) error {
	sl.BaseSeal = ub

	ad, err := bad.Encode(enc)
	if err != nil {
		return err
	}
	sl.ad = ad

	uci, err := DecodeConnInfo(bci, enc)
	if err != nil {
		return err
	}

	sl.ci = uci

	return nil
}
