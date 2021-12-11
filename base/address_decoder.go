package base

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

type AddressDecoder struct {
	ty hint.Type
	b  []byte
}

func (ad *AddressDecoder) Encode(enc encoder.Encoder) (Address, error) {
	var i Address
	err := encoder.DecodeWithHint(ad.b, enc, hint.NewHint(ad.ty, ""), &i)
	return i, err
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

	var i Address
	err = encoder.DecodeWithHint([]byte(p), enc, hint.NewHint(ty, ""), &i)
	return i, err
}
