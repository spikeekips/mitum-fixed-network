package key

import (
	stellarKeypair "github.com/stellar/go/keypair"

	"github.com/spikeekips/mitum/util/hint"
)

var (
	stellarPrivatekeyType = hint.MustNewType(0x01, 0x10, "stellar-privatekey")
	stellarPrivatekeyHint = hint.MustHint(stellarPrivatekeyType, "0.0.1")
	stellarPublickeyType  = hint.MustNewType(0x01, 0x11, "stellar-publickey")
	stellarPublickeyHint  = hint.MustHint(stellarPublickeyType, "0.0.1")
)

var (
	StellarPrivatekeyHinter = StellarPrivatekey{BaseKey: NewBaseKey(stellarPrivatekeyHint, nil)}
	StellarPublickeyHinter  = StellarPublickey{BaseKey: NewBaseKey(stellarPublickeyHint, nil)}
)

type StellarPrivatekey struct {
	BaseKey
	kp *stellarKeypair.Full
}

func newStellarPrivatekey(kp *stellarKeypair.Full) StellarPrivatekey {
	return StellarPrivatekey{
		BaseKey: NewBaseKey(stellarPrivatekeyHint, kp.Seed),
		kp:      kp,
	}
}

func NewStellarPrivatekey() (StellarPrivatekey, error) {
	full, err := stellarKeypair.Random()
	if err != nil {
		return StellarPrivatekey{}, err
	}

	return newStellarPrivatekey(full), nil
}

func NewStellarPrivatekeyFromString(s string) (StellarPrivatekey, error) {
	kp, err := stellarKeypair.Parse(s)
	if err != nil {
		return StellarPrivatekey{}, err
	}

	full, ok := kp.(*stellarKeypair.Full)
	if !ok {
		return StellarPrivatekey{}, InvalidKeyError.Errorf("not stellar private key; type=%T", kp)
	}

	return newStellarPrivatekey(full), nil
}

func (sp StellarPrivatekey) IsValid([]byte) error {
	if sp.kp == nil {
		return InvalidKeyError.Errorf("empty stellar Privatekey")
	}

	if kp, err := stellarKeypair.Parse(sp.kp.Seed()); err != nil {
		return InvalidKeyError.Wrap(err)
	} else if _, ok := kp.(*stellarKeypair.Full); !ok {
		return InvalidKeyError.Errorf("not stellar private key; type=%T", kp)
	}

	return nil
}

func (sp StellarPrivatekey) Equal(key Key) bool {
	if sp.kp == nil {
		return false
	}

	if sp.Hint().Type() != key.Hint().Type() {
		return false
	}

	k, ok := key.(StellarPrivatekey)
	if !ok {
		return false
	} else if k.kp == nil {
		return false
	}

	return sp.kp.Seed() == k.kp.Seed()
}

func (sp StellarPrivatekey) Publickey() Publickey {
	return newStellarPublickey(interface{}(sp.kp).(stellarKeypair.KP))
}

func (sp StellarPrivatekey) Sign(input []byte) (Signature, error) {
	sig, err := sp.kp.Sign(input)
	if err != nil {
		return nil, err
	}

	return Signature(sig), nil
}

func (sp *StellarPrivatekey) UnmarshalText(b []byte) error {
	if k, err := NewStellarPrivatekeyFromString(string(b)); err != nil {
		return err
	} else {
		*sp = k

		return nil
	}
}

type StellarPublickey struct {
	BaseKey
	kp stellarKeypair.KP
}

func newStellarPublickey(kp stellarKeypair.KP) StellarPublickey {
	return StellarPublickey{
		kp:      kp,
		BaseKey: NewBaseKey(stellarPublickeyHint, kp.Address),
	}
}

func NewStellarPublickeyFromString(s string) (StellarPublickey, error) {
	kp, err := stellarKeypair.ParseAddress(s)
	if err != nil {
		return StellarPublickey{}, nil
	}

	return newStellarPublickey(kp), nil
}

func (sp StellarPublickey) IsValid([]byte) error {
	if sp.kp == nil {
		return InvalidKeyError.Errorf("empty stellar Publickey")
	}

	return nil
}

func (sp StellarPublickey) Verify(input []byte, sig Signature) error {
	if err := sp.kp.Verify(input, []byte(sig)); err != nil {
		return SignatureVerificationFailedError.Wrap(err)
	}

	return nil
}

func (sp *StellarPublickey) UnmarshalText(b []byte) error {
	if k, err := NewStellarPublickeyFromString(string(b)); err != nil {
		return err
	} else {
		*sp = k

		return nil
	}
}
