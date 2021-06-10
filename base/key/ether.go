package key

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"

	etherCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spikeekips/mitum/util/hint"
)

var (
	EtherPrivatekeyType = hint.Type("ether-priv")
	EtherPrivatekeyHint = hint.NewHint(EtherPrivatekeyType, "v0.0.1")
	EtherPublickeyType  = hint.Type("ether-pub")
	EtherPublickeyHint  = hint.NewHint(EtherPublickeyType, "v0.0.1")
)

var (
	EtherPrivatekeyHinter = EtherPrivatekey{BaseKey: NewBaseKey(EtherPrivatekeyHint, nil)}
	EtherPublickeyHinter  = EtherPublickey{BaseKey: NewBaseKey(EtherPublickeyHint, nil)}
)

type EtherPrivatekey struct {
	BaseKey
	pk *ecdsa.PrivateKey
}

func newEtherPrivatekey(pk *ecdsa.PrivateKey) EtherPrivatekey {
	return EtherPrivatekey{
		BaseKey: NewBaseKey(EtherPrivatekeyHint, func() string {
			return hex.EncodeToString(etherCrypto.FromECDSA(pk))
		}),
		pk: pk,
	}
}

func NewEtherPrivatekey() (EtherPrivatekey, error) {
	pk, err := etherCrypto.GenerateKey()
	if err != nil {
		return EtherPrivatekey{}, err
	}

	return newEtherPrivatekey(pk), nil
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

	return newEtherPrivatekey(pk), nil
}

func (ep EtherPrivatekey) IsValid([]byte) error {
	if ep.pk == nil {
		return InvalidKeyError.Errorf("empty ether Privatekey")
	}

	return nil
}

func (ep EtherPrivatekey) Publickey() Publickey {
	return newEtherPublickey(&ep.pk.PublicKey)
}

func (ep EtherPrivatekey) Sign(input []byte) (Signature, error) {
	h := sha256.Sum256(input)
	r, s, err := ecdsa.Sign(rand.Reader, ep.pk, h[:])
	if err != nil {
		return nil, err
	}

	bs := make([]byte, 4+len(r.Bytes())+len(s.Bytes()))
	binary.LittleEndian.PutUint32(bs, uint32(len(r.Bytes())))

	copy(bs[4:], r.Bytes())
	copy(bs[4+len(r.Bytes()):], s.Bytes())

	return Signature(bs), nil
}

func (ep *EtherPrivatekey) UnmarshalText(b []byte) error {
	k, err := NewEtherPrivatekeyFromString(string(b))
	if err != nil {
		return err
	}

	*ep = k

	return nil
}

type EtherPublickey struct {
	BaseKey
	pk *ecdsa.PublicKey
}

func newEtherPublickey(pk *ecdsa.PublicKey) EtherPublickey {
	return EtherPublickey{
		BaseKey: NewBaseKey(EtherPublickeyHint, func() string {
			return hex.EncodeToString(etherCrypto.FromECDSAPub(pk))
		}),
		pk: pk,
	}
}

func NewEtherPublickeyFromString(s string) (EtherPublickey, error) {
	h, err := hex.DecodeString(s)
	if err != nil {
		return EtherPublickey{}, err
	}

	pk, err := etherCrypto.UnmarshalPubkey(h)
	if err != nil {
		return EtherPublickey{}, err
	}

	return newEtherPublickey(pk), nil
}

func (ep EtherPublickey) IsValid([]byte) error {
	if ep.pk == nil {
		return InvalidKeyError.Errorf("empty ether Publickey")
	}

	return nil
}

func (ep EtherPublickey) Verify(input []byte, sig Signature) error {
	if len(sig) < 4 {
		return SignatureVerificationFailedError.Errorf("invalid signature: length=%d signature=%x", len(sig), sig)
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

func (ep *EtherPublickey) UnmarshalText(b []byte) error {
	k, err := NewEtherPublickeyFromString(string(b))
	if err != nil {
		return err
	}

	*ep = k

	return nil
}
