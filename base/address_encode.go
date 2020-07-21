package base

import (
	"github.com/spikeekips/mitum/util/encoder"
	"golang.org/x/xerrors"
)

type AddressDecoder struct {
	encoder.HintedString
}

func (ad *AddressDecoder) Encode(enc encoder.Encoder) (Address, error) {
	if hinter, err := ad.HintedString.Encode(enc); err != nil {
		return nil, err
	} else if a, ok := hinter.(Address); !ok {
		return nil, xerrors.Errorf("not Address, %T", hinter)
	} else {
		return a, nil
	}
}
