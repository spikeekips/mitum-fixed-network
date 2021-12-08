package key

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (k BasePrivatekey) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

func (k *BasePrivatekey) UnmarshalText(b []byte) error {
	uk, err := ParseBasePrivatekey(string(b))
	if err != nil {
		return err
	}

	*k = uk

	return nil
}

func (k *BasePrivatekey) UnpackJSON(b []byte, _ *jsonenc.Encoder) error {
	uk, err := LoadBasePrivatekey(string(b))
	if err != nil {
		return err
	}

	*k = uk

	return nil
}

func (k BasePublickey) MarshalText() ([]byte, error) {
	return []byte(k.String()), nil
}

func (k *BasePublickey) UnmarshalText(b []byte) error {
	uk, err := ParseBasePublickey(string(b))
	if err != nil {
		return err
	}

	*k = uk

	return nil
}

func (k *BasePublickey) UnpackJSON(b []byte, _ *jsonenc.Encoder) error {
	uk, err := LoadBasePublickey(string(b))
	if err != nil {
		return err
	}

	*k = uk

	return nil
}
