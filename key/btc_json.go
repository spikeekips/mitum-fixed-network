package key // nolint

import "github.com/spikeekips/mitum/encoder"

func (bt BTCPrivatekey) PackJSON(enc *encoder.JSONEncoder) (interface{}, error) {
	return PackKeyJSON(bt, enc)
}

func (bt *BTCPrivatekey) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	s, err := UnpackKeyJSON(b, enc)
	if err != nil {
		return err
	}

	kp, err := NewBTCPrivatekeyFromString(s)
	if err != nil {
		return err
	}

	bt.wif = kp.wif

	return nil
}

func (bt BTCPublickey) PackJSON(enc *encoder.JSONEncoder) (interface{}, error) {
	return PackKeyJSON(bt, enc)
}

func (bt *BTCPublickey) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	s, err := UnpackKeyJSON(b, enc)
	if err != nil {
		return err
	}

	kp, err := NewBTCPublickeyFromString(s)
	if err != nil {
		return err
	}

	bt.pk = kp.pk

	return nil
}
