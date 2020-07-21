package key

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/hint"
)

func DecodeKey(enc encoder.Encoder, s string) (Key, error) {
	h, us, err := hint.ParseHintedString(s)
	if err != nil {
		return nil, err
	}

	kd := encoder.NewHintedString(h, us)
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

type PrivatekeyDecoder struct {
	encoder.HintedString
}

func (kd *PrivatekeyDecoder) Encode(enc encoder.Encoder) (Privatekey, error) {
	if hinter, err := kd.HintedString.Encode(enc); err != nil {
		return nil, err
	} else if k, ok := hinter.(Privatekey); !ok {
		return nil, xerrors.Errorf("not Privatekey, %T", hinter)
	} else {
		return k, nil
	}
}

type PublickeyDecoder struct {
	encoder.HintedString
}

func (kd *PublickeyDecoder) Encode(enc encoder.Encoder) (Publickey, error) {
	if hinter, err := kd.HintedString.Encode(enc); err != nil {
		return nil, err
	} else if k, ok := hinter.(Publickey); !ok {
		return nil, xerrors.Errorf("not Publickey, %T", hinter)
	} else {
		return k, nil
	}
}
