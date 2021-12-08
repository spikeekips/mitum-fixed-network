package key

import (
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil/base58"
	"github.com/spikeekips/mitum/util/hint"
)

type BasePublickey struct {
	k *btcec.PublicKey
	s string
	b []byte
}

func NewBasePublickey(k *btcec.PublicKey) BasePublickey {
	s := fmt.Sprintf("%s%s", base58.Encode(k.SerializeCompressed()), BasePublickeyType)
	return BasePublickey{
		k: k,
		s: s,
		b: []byte(s),
	}
}

func ParseBasePublickey(s string) (BasePublickey, error) {
	t := string(BasePublickeyType)
	switch {
	case !strings.HasSuffix(s, t):
		return BasePublickey{}, InvalidKeyError.Errorf("unknown publickey string")
	case len(s) <= len(t):
		return BasePublickey{}, InvalidKeyError.Errorf("invalid publickey string; too short")
	}

	return LoadBasePublickey(s[:len(s)-len(t)])
}

func LoadBasePublickey(s string) (BasePublickey, error) {
	k, err := btcec.ParsePubKey(base58.Decode(s), btcec.S256())
	if err != nil {
		return BasePublickey{}, InvalidKeyError.Wrap(err)
	}

	return NewBasePublickey(k), nil
}

func (k BasePublickey) String() string {
	return k.s
}

func (k BasePublickey) Bytes() []byte {
	return k.b
}

func (BasePublickey) Hint() hint.Hint {
	return BasePublickeyHint
}

func (k BasePublickey) IsValid([]byte) error {
	switch {
	case k.k == nil:
		return InvalidKeyError.Errorf("empty btc PublicKey")
	case len(k.s) < 1:
		return InvalidKeyError.Errorf("empty publickey string")
	case len(k.b) < 1:
		return InvalidKeyError.Errorf("empty publickey []byte")
	}

	return nil
}

func (k BasePublickey) Equal(b Key) bool {
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

func (k BasePublickey) Verify(input []byte, sig Signature) error {
	signature, err := btcec.ParseSignature(sig, btcec.S256())
	if err != nil {
		return err
	}

	if !signature.Verify(chainhash.DoubleHashB(input), k.k) {
		return SignatureVerificationFailedError.Call()
	}

	return nil
}
