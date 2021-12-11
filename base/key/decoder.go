package key

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

// KeyDecoder is basic unmarshaler for Keys. It can unmarshal Privatekey and
// publickey.
type KeyDecoder struct {
	ty hint.Type
	b  []byte
}

// Encode is used with encoder.Encoder.
// var de struct {
// 	Key KeyDecoder
// }
//
// err := json.Unmarshal(b, &de)
// if err != nil {
// 	return err
// }
//
// enc := jsonenc.NewEncoder()
// uk, err := de.Key.Encode(enc)
// if err != nil {
// 	return err
// }
//
// priv, ok := uk.(Privtekey)
// pub, ok := uk.(Publickey)
func (kd *KeyDecoder) Encode(enc encoder.Encoder) (Key, error) {
	if len(kd.b) < 1 {
		return nil, nil
	}

	return decodeKey(kd.b, kd.ty, enc)
}

func (kd KeyDecoder) Type() hint.Type {
	return kd.ty
}

func (kd KeyDecoder) Body() []byte {
	return kd.b
}

// PrivatekeyDecoder is basic unmarshaler for Privatekey.
type PrivatekeyDecoder struct {
	KeyDecoder
}

func (kd *PrivatekeyDecoder) Encode(enc encoder.Encoder) (Privatekey, error) {
	k, err := kd.KeyDecoder.Encode(enc)
	switch {
	case err != nil:
		return nil, err
	case k == nil:
		return nil, nil
	}

	priv, ok := k.(Privatekey)
	if !ok {
		return nil, errors.Errorf("not Privatekey: %T", k)
	}

	return priv, nil
}

// PublickeyDecoder is basic unmarshaler for Privatekey.
type PublickeyDecoder struct {
	KeyDecoder
}

func (kd *PublickeyDecoder) Encode(enc encoder.Encoder) (Publickey, error) {
	k, err := kd.KeyDecoder.Encode(enc)
	switch {
	case err != nil:
		return nil, err
	case k == nil:
		return nil, nil
	}

	pub, ok := k.(Publickey)
	if !ok {
		return nil, errors.Errorf("not Publickey: %T", k)
	}

	return pub, nil
}

// DecodeKeyFromString parses and decodes Key from string.
func DecodeKeyFromString(s string, enc encoder.Encoder) (Key, error) {
	if len(s) < 1 {
		return nil, nil
	}

	p, ty, err := hint.ParseFixedTypedString(s, KeyTypeSize)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode key, %q", s)
	}

	return decodeKey([]byte(p), ty, enc)
}

func DecodePrivatekeyFromString(s string, enc encoder.Encoder) (Privatekey, error) {
	k, err := DecodeKeyFromString(s, enc)
	switch {
	case err != nil:
		return nil, errors.Wrapf(err, "failed to decode privatekey, %q", s)
	case k == nil:
		return nil, nil
	}

	priv, ok := k.(Privatekey)
	if !ok {
		return nil, errors.Errorf("not privatekey: %T", k)
	}

	return priv, nil
}

func DecodePublickeyFromString(s string, enc encoder.Encoder) (Publickey, error) {
	k, err := DecodeKeyFromString(s, enc)
	switch {
	case err != nil:
		return nil, errors.Wrapf(err, "failed to decode publickey, %q", s)
	case k == nil:
		return nil, nil
	}

	pub, ok := k.(Publickey)
	if !ok {
		return nil, errors.Errorf("not publickey: %T", k)
	}

	return pub, nil
}

func decodeKey(b []byte, ty hint.Type, enc encoder.Encoder) (Key, error) {
	var k Key
	err := encoder.DecodeWithHint(b, enc, hint.NewHint(ty, ""), &k)
	return k, err
}
