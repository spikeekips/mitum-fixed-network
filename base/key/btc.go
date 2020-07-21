package key

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"

	"github.com/spikeekips/mitum/util/hint"
)

var (
	btcPrivatekeyType = hint.MustNewType(0x01, 0x12, "btc-privatekey")
	btcPrivatekeyHint = hint.MustHint(btcPrivatekeyType, "0.0.1")
	btcPublickeyType  = hint.MustNewType(0x01, 0x13, "btc-publickey")
	btcPublickeyHint  = hint.MustHint(btcPublickeyType, "0.0.1")
)

var (
	BTCPrivatekeyHinter = BTCPrivatekey{BaseKey: NewBaseKey(btcPrivatekeyHint, nil)}
	BTCPublickeyHinter  = BTCPublickey{BaseKey: NewBaseKey(btcPublickeyHint, nil)}
)

type BTCPrivatekey struct {
	BaseKey
	wif *btcutil.WIF
}

func newBTCPrivatekey(wif *btcutil.WIF) BTCPrivatekey {
	return BTCPrivatekey{
		wif:     wif,
		BaseKey: NewBaseKey(btcPrivatekeyHint, wif.String),
	}
}

func NewBTCPrivatekey() (BTCPrivatekey, error) {
	secret, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return BTCPrivatekey{}, err
	}

	wif, err := btcutil.NewWIF(secret, &chaincfg.MainNetParams, true)
	if err != nil {
		return BTCPrivatekey{}, err
	}

	return newBTCPrivatekey(wif), nil
}

func NewBTCPrivatekeyFromString(s string) (BTCPrivatekey, error) {
	wif, err := btcutil.DecodeWIF(s)
	if err != nil {
		return BTCPrivatekey{}, err
	}
	if !wif.IsForNet(&chaincfg.MainNetParams) {
		return BTCPrivatekey{}, InvalidKeyError.Errorf("unsupported BTC network")
	}

	return newBTCPrivatekey(wif), nil
}

func (bt BTCPrivatekey) IsValid([]byte) error {
	if bt.wif == nil {
		return InvalidKeyError.Errorf("empty btc wif")
	} else if bt.wif.PrivKey == nil {
		return InvalidKeyError.Errorf("empty btc wif.PrivKey")
	}

	return nil
}

func (bt BTCPrivatekey) Publickey() Publickey {
	return newBTCPublickey(bt.wif.PrivKey.PubKey())
}

func (bt BTCPrivatekey) Sign(input []byte) (Signature, error) {
	sig, err := bt.wif.PrivKey.Sign(chainhash.DoubleHashB(input))
	if err != nil {
		return nil, err
	}

	return Signature(sig.Serialize()), nil
}

func (bt *BTCPrivatekey) UnmarshalText(b []byte) error {
	if k, err := NewBTCPrivatekeyFromString(string(b)); err != nil {
		return err
	} else {
		*bt = k

		return nil
	}
}

type BTCPublickey struct {
	BaseKey
	pk *btcec.PublicKey
}

func newBTCPublickey(pk *btcec.PublicKey) BTCPublickey {
	return BTCPublickey{
		pk: pk,
		BaseKey: NewBaseKey(btcPublickeyHint, func() string {
			return base58.Encode(pk.SerializeCompressed())
		}),
	}
}

func NewBTCPublickeyFromString(s string) (BTCPublickey, error) {
	pk, err := btcec.ParsePubKey(base58.Decode(s), btcec.S256())
	if err != nil {
		return BTCPublickey{}, err
	}

	return BTCPublickey{
		BaseKey: NewBaseKey(btcPublickeyHint, func() string {
			return base58.Encode(pk.SerializeCompressed())
		}),
		pk: pk,
	}, nil
}

func (bt BTCPublickey) IsValid([]byte) error {
	if bt.pk == nil {
		return InvalidKeyError.Errorf("empty btc PublicKey")
	}

	return nil
}

func (bt BTCPublickey) Verify(input []byte, sig Signature) error {
	signature, err := btcec.ParseSignature(sig, btcec.S256())
	if err != nil {
		return err
	}

	if !signature.Verify(chainhash.DoubleHashB(input), bt.pk) {
		return SignatureVerificationFailedError
	}

	return nil
}

func (bt *BTCPublickey) UnmarshalText(b []byte) error {
	if k, err := NewBTCPublickeyFromString(string(b)); err != nil {
		return err
	} else {
		*bt = k

		return nil
	}
}
