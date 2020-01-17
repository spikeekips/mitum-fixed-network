package key

import (
	"github.com/spikeekips/mitum/encoder"
)

func (sp StellarPrivatekey) PackBSON(_ *encoder.BSONEncoder) (interface{}, error) {
	return &struct {
		K string `bson:"key"`
	}{
		K: sp.String(),
	}, nil
}

func (sp *StellarPrivatekey) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
	var k struct {
		K string `bson:"key"`
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

func (sp StellarPublickey) PackBSON(_ *encoder.BSONEncoder) (interface{}, error) {
	return &struct {
		K string `bson:"key"`
	}{
		K: sp.String(),
	}, nil
}

func (sp *StellarPublickey) UnpackBSON(b []byte, enc *encoder.BSONEncoder) error {
	var k struct {
		K string `bson:"key"`
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
