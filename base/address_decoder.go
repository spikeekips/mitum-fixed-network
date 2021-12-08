package base

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

type AddressDecoder struct {
	ty hint.Type
	b  []byte
}

func (ad *AddressDecoder) Encode(enc encoder.Encoder) (Address, error) {
	if len(ad.b) < 1 {
		return nil, nil
	}

	return decodeAddress(ad.b, ad.ty, enc)
}

// DecodeAddressFromString parses and decodes Address from string.
func DecodeAddressFromString(s string, enc encoder.Encoder) (Address, error) {
	if len(s) < 1 {
		return nil, nil
	}

	p, ty, err := hint.ParseFixedTypedString(s, AddressTypeSize)
	if err != nil {
		return nil, err
	}

	return decodeAddress([]byte(p), ty, enc)
}

func decodeAddress(b []byte, ty hint.Type, enc encoder.Encoder) (Address, error) {
	hinter, err := enc.DecodeWithHint(b, hint.NewHint(ty, ""))
	if err != nil {
		return nil, err
	}

	k, ok := hinter.(Address)
	if !ok {
		return nil, errors.Errorf("not Address: %T", hinter)
	}

	return k, nil
}
