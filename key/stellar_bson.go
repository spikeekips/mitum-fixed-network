package key

import "github.com/spikeekips/mitum/encoder"

func (sp StellarPrivatekey) EncodeBSON(_ *encoder.HintEncoder) (interface{}, error) {
	return &struct {
		K string `bson:"key"`
	}{
		K: sp.String(),
	}, nil
}

func (sp *StellarPrivatekey) DecodeBSON(enc *encoder.HintEncoder, b []byte) error {
	var k struct {
		K string `bson:"key"`
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

func (sp StellarPublickey) EncodeBSON(_ *encoder.HintEncoder) (interface{}, error) {
	return &struct {
		K string `bson:"key"`
	}{
		K: sp.String(),
	}, nil
}

func (sp *StellarPublickey) DecodeBSON(enc *encoder.HintEncoder, b []byte) error {
	var k struct {
		K string `bson:"key"`
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
