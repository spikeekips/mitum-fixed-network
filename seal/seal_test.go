package seal

import (
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/keypair"
)

type testSeal struct {
	suite.Suite
}

func (t *testSeal) TestIsValid() {
	defer common.DebugPanic()

	body := NewSealBody("new", 33)
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
	body := NewSealBody("new", 33)
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

	body := NewSealBody("new", 33)
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
