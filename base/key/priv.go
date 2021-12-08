package key

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	BasePrivatekeyType = hint.Type("mpr")
	BasePrivatekeyHint = hint.NewHint(BasePrivatekeyType, "v0.0.1")
	BasePublickeyType  = hint.Type("mpu")
	BasePublickeyHint  = hint.NewHint(BasePublickeyType, "v0.0.1")
)

const MinSeedSize = 36

// BasePrivatekey is based on BTC Privatekey.
type BasePrivatekey struct {
	wif *btcutil.WIF
	pub BasePublickey
	s   string
	b   []byte
}

func NewBasePrivatekey() BasePrivatekey {
	secret, _ := btcec.NewPrivateKey(btcec.S256())

	wif, _ := btcutil.NewWIF(secret, &chaincfg.MainNetParams, true)

	return newBasePrivatekey(wif)
}

func NewBasePrivatekeyFromSeed(s string) (BasePrivatekey, error) {
	if l := len(s); l < MinSeedSize {
		return BasePrivatekey{}, isvalid.InvalidError.Errorf(
			"wrong seed for privatekey; too short, %d < %d", l, MinSeedSize)
	}

	k, err := ecdsa.GenerateKey(
		btcec.S256(),
		bytes.NewReader([]byte(valuehash.NewSHA256([]byte(s)).String())),
	)
	if err != nil {
		return BasePrivatekey{}, err
	}

	wif, err := btcutil.NewWIF((*btcec.PrivateKey)(k), &chaincfg.MainNetParams, true)
	if err != nil {
		return BasePrivatekey{}, err
	}

	return newBasePrivatekey(wif), nil
}

func ParseBasePrivatekey(s string) (BasePrivatekey, error) {
	t := string(BasePrivatekeyType)
	switch {
	case !strings.HasSuffix(s, t):
		return BasePrivatekey{}, InvalidKeyError.Errorf("unknown privatekey string")
	case len(s) <= len(t):
		return BasePrivatekey{}, InvalidKeyError.Errorf("invalid privatekey string; too short")
	}

	return LoadBasePrivatekey(s[:len(s)-len(t)])
}

func LoadBasePrivatekey(s string) (BasePrivatekey, error) {
	wif, err := btcutil.DecodeWIF(s)
	if err != nil {
		return BasePrivatekey{}, InvalidKeyError.Wrap(err)
	}

	return newBasePrivatekey(wif), nil
}

func newBasePrivatekey(wif *btcutil.WIF) BasePrivatekey {
	s := fmt.Sprintf("%s%s", wif.String(), BasePrivatekeyType)
	pub := NewBasePublickey(wif.PrivKey.PubKey())

	return BasePrivatekey{wif: wif, s: s, b: []byte(s), pub: pub}
}

func (BasePrivatekey) Hint() hint.Hint {
	return BasePrivatekeyHint
}

func (k BasePrivatekey) Publickey() Publickey {
	return k.pub
}

func (k BasePrivatekey) Equal(b Key) bool {
	if b == nil {
		return false
	}

	if k.Hint().Type() != b.Hint().Type() {
		return false
	}

	if err := b.IsValid(nil); err != nil {
		return false
	}

	return k.s == b.String()
}

func (k BasePrivatekey) String() string {
	return k.s
}

func (k BasePrivatekey) Bytes() []byte {
	return k.b
}

func (k BasePrivatekey) IsValid([]byte) error {
	switch {
	case k.wif == nil:
		return InvalidKeyError.Errorf("empty btc wif")
	case k.wif.PrivKey == nil:
		return InvalidKeyError.Errorf("empty btc wif.PrivKey")
	case len(k.s) < 1:
		return InvalidKeyError.Errorf("empty privatekey string")
	case len(k.b) < 1:
		return InvalidKeyError.Errorf("empty privatekey []byte")
	}

	return nil
}

func (k BasePrivatekey) Sign(b []byte) (Signature, error) {
	sig, err := k.wif.PrivKey.Sign(chainhash.DoubleHashB(b))
	if err != nil {
		return nil, err
	}

	return Signature(sig.Serialize()), nil
}
