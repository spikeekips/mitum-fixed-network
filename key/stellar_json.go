package key

import "github.com/spikeekips/mitum/encoder"

func (sp StellarPrivatekey) PackJSON(_ *encoder.JSONEncoder) (interface{}, error) {
	return &struct {
		encoder.JSONPackHintedHead
		K string `json:"key"`
	}{
		K: sp.String(),
	}, nil
}

func (sp *StellarPrivatekey) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var k struct {
		K string `json:"key"`
	}
	if err := enc.Unmarshal(b, &k); err != nil {
		return err
	}

	kp, err := NewStellarPrivatekeyFromString(k.K)
	if err != nil {
		return err
	}

	sp.kp = kp.kp

	return nil
}

func (sp StellarPublickey) PackJSON(_ *encoder.JSONEncoder) (interface{}, error) {
	return &struct {
		encoder.JSONPackHintedHead
		K string `json:"key"`
	}{
		K: sp.String(),
	}, nil
}

func (sp *StellarPublickey) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var k struct {
		K string `json:"key"`
	}
	if err := enc.Unmarshal(b, &k); err != nil {
		return err
	}

	kp, err := NewStellarPublickeyFromString(k.K)
	if err != nil {
		return err
	}

	sp.kp = kp.kp

	return nil
}
