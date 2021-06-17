package base

import (
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (vf *BaseVoteproofNodeFact) unpack(
	enc encoder.Encoder,
	bAddress AddressDecoder,
	blt,
	fact valuehash.Hash,
	factSignature key.Signature,
	bSigner key.PublickeyDecoder,
) error {
	address, err := bAddress.Encode(enc)
	if err != nil {
		return err
	}

	signer, err := bSigner.Encode(enc)
	if err != nil {
		return err
	}

	vf.address = address
	vf.ballot = blt
	vf.fact = fact
	vf.factSignature = factSignature
	vf.signer = signer

	return nil
}
