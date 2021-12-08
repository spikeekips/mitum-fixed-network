package base

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

func (ad *AddressDecoder) UnmarshalJSON(b []byte) error {
	if jsonenc.NULL == string(b) {
		return nil
	}

	var s string
	if err := jsonenc.Unmarshal(b, &s); err != nil {
		return err
	}

	p, ty, err := hint.ParseFixedTypedString(s, AddressTypeSize)
	if err != nil {
		return err
	}

	ad.ty = ty
	ad.b = []byte(p)

	return nil
}
