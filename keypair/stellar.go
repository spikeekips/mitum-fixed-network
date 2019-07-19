package keypair

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
	stellarHash "github.com/stellar/go/hash"
	stellarKeypair "github.com/stellar/go/keypair"

	"github.com/spikeekips/mitum/common"
)

var (
	StellarType common.DataType = common.NewDataType(1, "stellar")
)

type Stellar struct {
}

func (s Stellar) Type() common.DataType {
	return StellarType
}

// New generates the new random keypair
func (s Stellar) New() (PrivateKey, error) {
	seed, err := stellarKeypair.Random()
	if err != nil {
		return nil, err
	}

	return StellarPrivateKey{kp: seed}, nil
}

// NewFromSeed generates the keypair from raw seed
func (s Stellar) NewFromSeed(b []byte) (PrivateKey, error) {
	seed, err := stellarKeypair.FromRawSeed(stellarHash.Hash(b))
	if err != nil {
		return nil, err
	}

	return StellarPrivateKey{kp: seed}, nil
}

type StellarPublicKey struct {
	kp stellarKeypair.KP
}

func (s StellarPublicKey) Type() common.DataType {
	return StellarType
}

func (s StellarPublicKey) Kind() Kind {
	return PublicKeyKind
}

func (s StellarPublicKey) Verify(input []byte, sig Signature) error {
	if err := s.kp.Verify(input, []byte(sig)); err != nil {
		return SignatureVerificationFailedError.New(err)
	}

	return nil
}

func (s StellarPublicKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s StellarPublicKey) String() string {
	return fmt.Sprintf("%s:%s:%s", s.kp.Address(), s.Kind(), s.Type())
}

func (s StellarPublicKey) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, struct {
		Type common.DataType
		Kind Kind
		Key  string
	}{
		Type: s.Type(),
		Kind: s.Kind(),
		Key:  s.kp.Address(),
	})
}

func (s *StellarPublicKey) DecodeRLP(st *rlp.Stream) error {
	var d struct {
		Type common.DataType
		Kind Kind
		Key  string
	}
	if err := st.Decode(&d); err != nil {
		return err
	}

	if !s.Type().Equal(d.Type) {
		return FailedToEncodeKeypairError.Newf("not stellar keypair type; type=%q", d.Type)
	}

	if s.Kind() != d.Kind {
		return FailedToEncodeKeypairError.Newf("not public type; kind=%q", d.Kind)
	}

	kp, err := stellarKeypair.Parse(d.Key)
	if err != nil {
		return err
	}

	s.kp = kp

	return nil
}

func (s StellarPublicKey) Equal(k Key) bool {
	if !s.Type().Equal(k.Type()) {
		return false
	}

	if s.Kind() != k.Kind() {
		return false
	}

	ks, ok := k.(StellarPublicKey)
	if !ok {
		return false
	}

	return s.kp.Address() == ks.kp.Address()
}

func (s StellarPublicKey) NativePublicKey() []byte {
	return []byte(s.kp.Address())
}

func (s StellarPublicKey) IsValid() error {
	if _, err := stellarKeypair.Parse(s.kp.Address()); err != nil {
		return err
	}

	return nil
}

type StellarPrivateKey struct {
	kp *stellarKeypair.Full
}

func NewStellarPrivateKey() (PrivateKey, error) {
	seed, err := stellarKeypair.Random()
	if err != nil {
		return nil, err
	}

	return StellarPrivateKey{kp: seed}, nil
}

func (s StellarPrivateKey) Type() common.DataType {
	return StellarType
}

func (s StellarPrivateKey) Kind() Kind {
	return PrivateKeyKind
}

func (s StellarPrivateKey) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s StellarPrivateKey) String() string {
	return fmt.Sprintf("%s:%s:%s", s.kp.Seed(), s.Kind(), s.Type())
}

func (s StellarPrivateKey) Sign(input []byte) (Signature, error) {
	sig, err := s.kp.Sign(input)
	if err != nil {
		return nil, err
	}

	return Signature(sig), nil
}

func (s StellarPrivateKey) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, struct {
		Type common.DataType
		Kind Kind
		Key  string
	}{
		Type: s.Type(),
		Kind: s.Kind(),
		Key:  s.kp.Seed(),
	})
}

func (s *StellarPrivateKey) DecodeRLP(st *rlp.Stream) error {
	var d struct {
		Type common.DataType
		Kind Kind
		Key  string
	}
	if err := st.Decode(&d); err != nil {
		return err
	}

	if !s.Type().Equal(d.Type) {
		return FailedToEncodeKeypairError.Newf("not stellar keypair type; type=%q", d.Type)
	}

	if s.Kind() != d.Kind {
		return FailedToEncodeKeypairError.Newf("not private; kind=%q", d.Kind)
	}

	kp, err := stellarKeypair.Parse(d.Key)
	if err != nil {
		return err
	} else if full, ok := kp.(*stellarKeypair.Full); !ok {
		return FailedToEncodeKeypairError.Newf("not private key; type=%T", kp)
	} else {
		s.kp = full
	}

	return nil
}

func (s StellarPrivateKey) Equal(k Key) bool {
	if !s.Type().Equal(k.Type()) {
		return false
	}

	if s.Kind() != k.Kind() {
		return false
	}

	ks, ok := k.(StellarPrivateKey)
	if !ok {
		return false
	}

	return s.kp.Seed() == ks.kp.Seed()
}

func (s StellarPrivateKey) PublicKey() PublicKey {
	return StellarPublicKey{kp: interface{}(s.kp).(stellarKeypair.KP)}
}

func (s StellarPrivateKey) NativePublicKey() []byte {
	return []byte(s.kp.Address())
}

func (s StellarPrivateKey) NativePrivateKey() []byte {
	return []byte(s.kp.Seed())
}

func (s StellarPrivateKey) IsValid() error {
	kp, err := stellarKeypair.Parse(s.kp.Seed())
	if err != nil {
		return err
	} else if _, ok := kp.(*stellarKeypair.Full); !ok {
		return FailedToEncodeKeypairError.Newf("not private key; type=%T", kp)
	}

	return nil
}
