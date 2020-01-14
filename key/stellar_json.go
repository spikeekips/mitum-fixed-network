package key

import "github.com/spikeekips/mitum/encoder"

func (sp StellarPrivatekey) EncodeJSON(_ *encoder.HintEncoder) (interface{}, error) {
	return &struct {
		encoder.JSONHinterHead
		K string `json:"key"`
	}{
		K: sp.String(),
	}, nil
}

func (sp *StellarPrivatekey) DecodeJSON(enc *encoder.HintEncoder, b []byte) error {
	var k struct {
		K string `json:"key"`
	}
	if err := enc.Encoder().Unmarshal(b, &k); err != nil {
		return err
	}

	kp, err := NewStellarPrivatekeyFromString(k.K)
	if err != nil {
		return err
	}

	sp.kp = kp.kp

	return nil
}

func (sp StellarPublickey) EncodeJSON(_ *encoder.HintEncoder) (interface{}, error) {
	return &struct {
		encoder.JSONHinterHead
		K string `json:"key"`
	}{
		K: sp.String(),
	}, nil
}

func (sp *StellarPublickey) DecodeJSON(enc *encoder.HintEncoder, b []byte) error {
	var k struct {
		K string `json:"key"`
	}
	if err := enc.Encoder().Unmarshal(b, &k); err != nil {
		return err
	}

	kp, err := NewStellarPublickeyFromString(k.K)
	if err != nil {
		return err
	}

	sp.kp = kp.kp

	return nil
}
