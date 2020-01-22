package key

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"

	etherCrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/spikeekips/mitum/hint"
)

var etherPrivatekeyHint = hint.MustHint(hint.Type([2]byte{0x02, 0x04}), "0.1")
var etherPublickeyHint = hint.MustHint(hint.Type([2]byte{0x02, 0x05}), "0.1")

type EtherPrivatekey struct {
	pk *ecdsa.PrivateKey
}

func NewEtherPrivatekey() (EtherPrivatekey, error) {
	pk, err := etherCrypto.GenerateKey()
	if err != nil {
		return EtherPrivatekey{}, err
	}

	return EtherPrivatekey{pk: pk}, nil
}

func NewEtherPrivatekeyFromString(s string) (EtherPrivatekey, error) {
	h, err := hex.DecodeString(s)
	if err != nil {
		return EtherPrivatekey{}, err
	}

	pk, err := etherCrypto.ToECDSA(h)
	if err != nil {
		return EtherPrivatekey{}, err
	}

	return EtherPrivatekey{pk: pk}, nil
}

func (ep EtherPrivatekey) String() string {
	if ep.pk == nil {
		return ""
	}

	return hex.EncodeToString(etherCrypto.FromECDSA(ep.pk))
}

func (ep EtherPrivatekey) Hint() hint.Hint {
	return etherPrivatekeyHint
}

func (ep EtherPrivatekey) IsValid([]byte) error {
	if ep.pk == nil {
		return InvalidKeyError.Wrapf("empty ether Privatekey")
	}

	return nil
}

func (ep EtherPrivatekey) Equal(key Key) bool {
	if ep.pk == nil {
		return false
	}

	if ep.Hint().Type() != key.Hint().Type() {
		return false
	}

	k, ok := key.(EtherPrivatekey)
	if !ok {
		return false
	}

	return bytes.Equal(
		etherCrypto.FromECDSA(ep.pk),
		etherCrypto.FromECDSA(k.pk),
	)
}

func (ep EtherPrivatekey) Publickey() Publickey {
	return EtherPublickey{pk: &ep.pk.PublicKey}
}

func (ep EtherPrivatekey) Sign(input []byte) (Signature, error) {
	h := sha256.Sum256(input)
	r, s, err := ecdsa.Sign(rand.Reader, ep.pk, h[:])
	if err != nil {
		return nil, err
	}

	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, uint32(len(r.Bytes())))

	bs = append(bs, r.Bytes()...)
	bs = append(bs, s.Bytes()...)

	return Signature(bs), nil
}

type EtherPublickey struct {
	pk *ecdsa.PublicKey
}

func NewEtherPublickey(s string) (EtherPublickey, error) {
	h, err := hex.DecodeString(s)
	if err != nil {
		return EtherPublickey{}, err
	}

	pk, err := etherCrypto.UnmarshalPubkey(h)
	if err != nil {
		return EtherPublickey{}, err
	}

	return EtherPublickey{pk: pk}, nil
}

func (ep EtherPublickey) String() string {
	if ep.pk == nil {
		return ""
	}

	return hex.EncodeToString(etherCrypto.FromECDSAPub(ep.pk))
}

func (ep EtherPublickey) Hint() hint.Hint {
	return etherPublickeyHint
}

func (ep EtherPublickey) IsValid([]byte) error {
	if ep.pk == nil {
		return InvalidKeyError.Wrapf("empty ether Publickey")
	}

	return nil
}

func (ep EtherPublickey) Equal(key Key) bool {
	if ep.pk == nil {
		return false
	}

	if ep.Hint().Type() != key.Hint().Type() {
		return false
	}

	k, ok := key.(EtherPublickey)
	if !ok {
		return false
	}

	return bytes.Equal(
		etherCrypto.FromECDSAPub(ep.pk),
		etherCrypto.FromECDSAPub(k.pk),
	)
}

func (ep EtherPublickey) Verify(input []byte, sig Signature) error {
	if len(sig) < 4 {
		return SignatureVerificationFailedError.Wrapf("invalid signature: length=%d signature=%x", len(sig), sig)
	}
	rlength := int(binary.LittleEndian.Uint32(sig[:4]))

	r := big.NewInt(0).SetBytes(sig[4 : 4+rlength])
	s := big.NewInt(0).SetBytes(sig[4+rlength:])

	h := sha256.Sum256(input)
	if !ecdsa.Verify(ep.pk, h[:], r, s) {
		return SignatureVerificationFailedError
	}

	return nil
}
