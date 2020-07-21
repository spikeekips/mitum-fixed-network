package key

import (
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
	"golang.org/x/xerrors"
)

type KeyDecoder struct {
	h hint.Hint
	s string
}

func (kd KeyDecoder) Hint() hint.Hint {
	return kd.h
}

func (kd KeyDecoder) StringValue() string {
	return kd.s
}

func (kd KeyDecoder) IsValid([]byte) error {
	if err := kd.h.IsValid(nil); err != nil {
		return InvalidKeyError.Wrap(err)
	}

	if len(kd.s) < 1 {
		return InvalidKeyError.Errorf("empty source string for KeyDecoder")
	}

	return nil
}

func (kd KeyDecoder) Encode(enc encoder.Encoder) (Key, error) {
	if hinter, err := enc.DecodeWithHint(kd.h, []byte(kd.s)); err != nil {
		return nil, err
	} else {
		return hinter.(Key), nil
	}
}

func DecodeKey(enc encoder.Encoder, s string) (Key, error) {
	h, us, err := hint.ParseHintedString(s)
	if err != nil {
		return nil, err
	}

	kd := KeyDecoder{h: h, s: us}
	if k, err := kd.Encode(enc); err != nil {
		return nil, err
	} else if pk, ok := k.(Key); !ok {
		return nil, xerrors.Errorf("not key.Key; type=%T", k)
	} else {
		return pk, nil
	}
}

func DecodePrivatekey(enc encoder.Encoder, s string) (Privatekey, error) {
	if k, err := DecodeKey(enc, s); err != nil {
		return nil, err
	} else if pk, ok := k.(Privatekey); !ok {
		return nil, xerrors.Errorf("not key.Privatekey; type=%T", k)
	} else {
		return pk, nil
	}
}

func DecodePublickey(enc encoder.Encoder, s string) (Publickey, error) {
	if k, err := DecodeKey(enc, s); err != nil {
		return nil, err
	} else if pk, ok := k.(Publickey); !ok {
		return nil, xerrors.Errorf("not key.Publickey; type=%T", k)
	} else {
		return pk, nil
	}
}
