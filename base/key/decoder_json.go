package key

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

func (kd *KeyDecoder) UnmarshalJSON(b []byte) error {
	if jsonenc.NULL == string(b) {
		return nil
	}

	var s string
	if err := jsonenc.Unmarshal(b, &s); err != nil {
		return err
	}

	p, ty, err := hint.ParseFixedTypedString(s, KeyTypeSize)
	if err != nil {
		return err
	}

	kd.ty = ty
	kd.b = []byte(p)

	return nil
}
