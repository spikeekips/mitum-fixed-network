package seal

import (
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/keypair"
)

type tSealBody struct {
	hash hash.Hash
	T    common.DataType
	A    string
	B    uint64
}

func (t tSealBody) Type() common.DataType {
	return t.T
}

func (t tSealBody) makeHash() (hash.Hash, error) {
	b, err := rlp.EncodeToBytes(t)
	if err != nil {
		log.Error("Hash() failed", "error", err)
		return hash.Hash{}, err
	}

	h, err := hash.NewDoubleSHAHash("ts", b)
	if err != nil {
		log.Error("Hash() failed", "error", err)
		return hash.Hash{}, err
	}

	return h, nil
}

func (t tSealBody) Hash() hash.Hash {
	return t.hash
}

func (t tSealBody) IsValid() error {
	if t.T.Empty() {
		return xerrors.Errorf("empty T")
	}

	if len(t.A) < 1 {
		return xerrors.Errorf("empty A")
	}

	if t.B < 1 {
		return xerrors.Errorf("negative B")
	}

	return nil
}

func (t tSealBody) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, struct {
		H hash.Hash
		T common.DataType
		A string
		B uint64
	}{
		H: t.hash,
		T: t.T,
		A: t.A,
		B: t.B,
	})
}

func (t *tSealBody) DecodeRLP(s *rlp.Stream) error {
	var h struct {
		H hash.Hash
		T common.DataType
		A string
		B uint64
	}

	if err := s.Decode(&h); err != nil {
		return err
	}

	t.hash = h.H
	t.T = h.T
	t.A = h.A
	t.B = h.B

	return nil
}

func newSealBody(a string, b uint64) tSealBody {
	dt := common.NewDataType(33, "test-seal-body")
	ts := tSealBody{T: dt, A: a, B: b}
	ts.hash, _ = ts.makeHash()
	return ts
}

func newSealBodySigned(pk keypair.PrivateKey, a string, b uint64) (Seal, error) {
	body := newSealBody(a, b)
	sl := NewBaseSeal(body)
	if err := sl.Sign(pk, []byte{}); err != nil {
		return nil, err
	}

	return sl, nil
}

type testSeal struct {
	suite.Suite
}

func (t *testSeal) TestIsValid() {
	defer common.DebugPanic()

	body := newSealBody("new", 33)
	sl := NewBaseSeal(body)

	{ // before signing, seal is invalid
		err := sl.IsValid()
		t.True(xerrors.Is(InvalidSealError, err))
	}

	// signing
	pk, _ := keypair.NewStellarPrivateKey()
	err := sl.Sign(pk, []byte{})
	t.NoError(err)

	err = sl.IsValid()
	t.NoError(err)
}

func (t *testSeal) TestSign() {
	body := newSealBody("new", 33)
	sl := NewBaseSeal(body)

	// signing
	salt := []byte("salt")

	pk, _ := keypair.NewStellarPrivateKey()

	err := sl.Sign(pk, salt)
	t.NoError(err)
	t.NotEmpty(sl.Signature())

	err = sl.CheckSignature(salt)
	t.NoError(err)
}

func (t *testSeal) TestEncode() {
	defer common.DebugPanic()

	body := newSealBody("new", 33)
	sl := NewBaseSeal(body)

	{ // before signing; encoding will be failed
		_, err := rlp.EncodeToBytes(sl)
		t.True(xerrors.Is(InvalidSealError, err))
	}

	// signing
	pk, _ := keypair.NewStellarPrivateKey()
	err := sl.Sign(pk, []byte{})
	t.NoError(err)

	err = sl.IsValid()
	t.NoError(err)

	b, err := rlp.EncodeToBytes(sl)
	t.NoError(err)

	var decoded BaseSeal
	err = rlp.DecodeBytes(b, &decoded)
	t.NoError(err)

	// decoding does not understand Body, so Body is nil
	t.Nil(decoded.Body())

	err = decoded.IsValid()
	t.True(xerrors.Is(InvalidSealError, err))
}

func TestSeal(t *testing.T) {
	suite.Run(t, new(testSeal))
}
