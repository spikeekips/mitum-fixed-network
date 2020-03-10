package key

import (
	stellarKeypair "github.com/stellar/go/keypair"

	"github.com/spikeekips/mitum/hint"
)

var (
	stellarPrivatekeyHint = hint.MustHint(hint.Type{0x02, 0x00}, "0.0.1")
	stellarPublickeyHint  = hint.MustHint(hint.Type{0x02, 0x01}, "0.0.1")
)

type StellarPrivatekey struct {
	kp *stellarKeypair.Full
}

func NewStellarPrivatekey() (StellarPrivatekey, error) {
	full, err := stellarKeypair.Random()
	if err != nil {
		return StellarPrivatekey{}, err
	}

	return StellarPrivatekey{kp: full}, nil
}

func NewStellarPrivatekeyFromString(s string) (StellarPrivatekey, error) {
	kp, err := stellarKeypair.Parse(s)
	if err != nil {
		return StellarPrivatekey{}, err
	}

	full, ok := kp.(*stellarKeypair.Full)
	if !ok {
		return StellarPrivatekey{}, InvalidKeyError.Wrapf("not stellar private key; type=%T", kp)
	}

	return StellarPrivatekey{kp: full}, nil
}

func (sp StellarPrivatekey) String() string {
	if sp.kp == nil {
		return ""
	}

	return sp.kp.Seed()
}

func (sp StellarPrivatekey) Hint() hint.Hint {
	return stellarPrivatekeyHint
}

func (sp StellarPrivatekey) IsValid([]byte) error {
	if sp.kp == nil {
		return InvalidKeyError.Wrapf("empty stellar Privatekey")
	}

	if kp, err := stellarKeypair.Parse(sp.kp.Seed()); err != nil {
		return InvalidKeyError.Wrap(err)
	} else if _, ok := kp.(*stellarKeypair.Full); !ok {
		return InvalidKeyError.Wrapf("not stellar private key; type=%T", kp)
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
	return StellarPublickey{kp: interface{}(sp.kp).(stellarKeypair.KP)}
}

func (sp StellarPrivatekey) Sign(input []byte) (Signature, error) {
	sig, err := sp.kp.Sign(input)
	if err != nil {
		return nil, err
	}

	return Signature(sig), nil
}

type StellarPublickey struct {
	kp stellarKeypair.KP
}

func NewStellarPublickeyFromString(s string) (StellarPublickey, error) {
	addr, err := stellarKeypair.ParseAddress(s)
	if err != nil {
		return StellarPublickey{}, nil
	}

	return StellarPublickey{kp: addr}, nil
}

func (sp StellarPublickey) String() string {
	if sp.kp == nil {
		return ""
	}

	return sp.kp.Address()
}

func (sp StellarPublickey) Hint() hint.Hint {
	return stellarPublickeyHint
}

func (sp StellarPublickey) IsValid([]byte) error {
	if sp.kp == nil {
		return InvalidKeyError.Wrapf("empty stellar Publickey")
	}

	return nil
}

func (sp StellarPublickey) Equal(key Key) bool {
	if sp.kp == nil {
		return false
	}

	if sp.Hint().Type() != key.Hint().Type() {
		return false
	}

	k, ok := key.(StellarPublickey)
	if !ok {
		return false
	}

	return sp.kp.Address() == k.kp.Address()
}

func (sp StellarPublickey) Verify(input []byte, sig Signature) error {
	if err := sp.kp.Verify(input, []byte(sig)); err != nil {
		return SignatureVerificationFailedError.Wrap(err)
	}

	return nil
}
