package base

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util/encoder"
)

func (bn *BaseNodeV0) unpack(enc encoder.Encoder, bad []byte, bpk encoder.HintedString) error {
	var address Address
	if a, err := DecodeAddress(enc, bad); err != nil {
		return err
	} else {
		address = a
	}

	var pk key.Publickey
	if k, err := bpk.Encode(enc); err != nil {
		return err
	} else if p, ok := k.(key.Publickey); !ok {
		return xerrors.Errorf("not key.Publickey; type=%T", k)
	} else {
		pk = p
	}

	bn.address = address
	bn.publickey = pk

	return nil
}
