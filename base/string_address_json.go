package base

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (ad StringAddress) MarshalText() ([]byte, error) {
	return ad.Bytes(), nil
}

func (ad *StringAddress) UnpackJSON(b []byte, _ *jsonenc.Encoder) error {
	*ad = NewStringAddress(string(b))

	return nil
}
