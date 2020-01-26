package key // nolint

import "github.com/spikeekips/mitum/encoder"

func (sp StellarPrivatekey) PackJSON(enc *encoder.JSONEncoder) (interface{}, error) {
	return PackKeyJSON(sp, enc)
}

func (sp *StellarPrivatekey) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	s, err := UnpackKeyJSON(b, enc)
	if err != nil {
		return err
	}

	kp, err := NewStellarPrivatekeyFromString(s)
	if err != nil {
		return err
	}

	sp.kp = kp.kp

	return nil
}

func (sp StellarPublickey) PackJSON(enc *encoder.JSONEncoder) (interface{}, error) {
	return PackKeyJSON(sp, enc)
}

func (sp *StellarPublickey) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	s, err := UnpackKeyJSON(b, enc)
	if err != nil {
		return err
	}
	kp, err := NewStellarPublickeyFromString(s)
	if err != nil {
		return err
	}

	sp.kp = kp.kp

	return nil
}
