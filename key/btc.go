package key

import (
	"bytes"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/base58"

	"github.com/spikeekips/mitum/hint"
)

var btcPrivatekeyHint hint.Hint = hint.MustHint(hint.Type{0x02, 0x02}, "0.1")
var btcPublickeyHint hint.Hint = hint.MustHint(hint.Type{0x02, 0x03}, "0.1")

type BTCPrivatekey struct {
	wif *btcutil.WIF
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

	return BTCPrivatekey{wif: wif}, nil
}

func NewBTCPrivatekeyFromString(s string) (BTCPrivatekey, error) {
	wif, err := btcutil.DecodeWIF(s)
	if err != nil {
		return BTCPrivatekey{}, err
	}
	if !wif.IsForNet(&chaincfg.MainNetParams) {
		return BTCPrivatekey{}, InvalidKeyError.Wrapf("unsupported BTC network")
	}

	return BTCPrivatekey{wif: wif}, nil
}

func (bt BTCPrivatekey) String() string {
	return bt.wif.String()
}

func (bt BTCPrivatekey) Hint() hint.Hint {
	return btcPrivatekeyHint
}

func (bt BTCPrivatekey) IsValid([]byte) error {
	if bt.wif == nil {
		return InvalidKeyError.Wrapf("empty btc wif")
	} else if bt.wif.PrivKey == nil {
		return InvalidKeyError.Wrapf("empty btc wif.PrivKey")
	}

	return nil
}

func (bt BTCPrivatekey) Equal(key Key) bool {
	if bt.wif == nil || bt.wif.PrivKey == nil {
		return false
	}

	if bt.Hint().Type() != key.Hint().Type() {
		return false
	}

	k, ok := key.(BTCPrivatekey)
	if !ok {
		return false
	}

	return bytes.Equal(
		bt.wif.PrivKey.Serialize(),
		k.wif.PrivKey.Serialize(),
	)
}

func (bt BTCPrivatekey) Publickey() Publickey {
	return BTCPublickey{pk: bt.wif.PrivKey.PubKey()}
}

func (bt BTCPrivatekey) Sign(input []byte) (Signature, error) {
	sig, err := bt.wif.PrivKey.Sign(input)
	if err != nil {
		return nil, err
	}

	return Signature(sig.Serialize()), nil
}

type BTCPublickey struct {
	pk *btcec.PublicKey
}

func NewBTCPublickeyFromString(s string) (BTCPublickey, error) {
	pk, err := btcec.ParsePubKey(base58.Decode(s), btcec.S256())
	if err != nil {
		return BTCPublickey{}, err
	}

	return BTCPublickey{pk: pk}, nil
}

func (bt BTCPublickey) String() string {
	return base58.Encode(bt.pk.SerializeCompressed())
}

func (bt BTCPublickey) Hint() hint.Hint {
	return btcPublickeyHint
}

func (bt BTCPublickey) IsValid([]byte) error {
	if bt.pk == nil {
		return InvalidKeyError.Wrapf("empty btc PublicKey")
	}

	return nil
}

func (bt BTCPublickey) Equal(key Key) bool {
	if bt.pk == nil {
		return false
	}

	if bt.Hint().Type() != key.Hint().Type() {
		return false
	}

	k, ok := key.(BTCPublickey)
	if !ok {
		return false
	}

	return bt.pk.IsEqual(k.pk)
}

func (bt BTCPublickey) Verify(input []byte, sig Signature) error {
	signature, err := btcec.ParseSignature(sig, btcec.S256())
	if err != nil {
		return err
	}

	if !signature.Verify(input, bt.pk) {
		return SignatureVerificationFailedError
	}

	return nil
}
