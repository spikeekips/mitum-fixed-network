package base

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util/encoder"
)

type AddressDecoder struct {
	encoder.HintedString
}

func (ad *AddressDecoder) Encode(enc encoder.Encoder) (Address, error) {
	if err := ad.Hint().IsValid(nil); err != nil {
		return nil, nil // nolint:nilerr
	}

	if hinter, err := ad.HintedString.Decode(enc); err != nil {
		return nil, err
	} else if a, ok := hinter.(Address); !ok {
		return nil, errors.Errorf("not Address, %T", hinter)
	} else {
		return a, nil
	}
}
