package keypair

import (
	"regexp"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testStellarKeypair struct {
	suite.Suite
}

func (t *testStellarKeypair) TestNew() {
	st0, _ := Stellar{}.New()
	t.Equal(StellarType, st0.Type())

	st1, _ := Stellar{}.New()
	t.False(st0.Equal(st1))
}

func (t *testStellarKeypair) TestPublicKey() {
	st, _ := Stellar{}.New()
	pk := st.PublicKey()
	t.Equal(StellarType, pk.Type())
	t.NotEmpty(pk)
	t.True(pk.Equal(pk))
	t.Regexp(regexp.MustCompile(`^G`), pk.String())
}

func (t *testStellarKeypair) TestEncodeRLP() {
	pr, _ := Stellar{}.New()

	{
		b, err := rlp.EncodeToBytes(pr)
		t.NoError(err)
		t.NotEmpty(b)

		var key StellarPrivateKey
		err = rlp.DecodeBytes(b, &key)
		t.NoError(err)
		t.NotEmpty(key)

		t.Equal(PrivateKeyKind, key.Kind())
		t.True(pr.Equal(key))
	}

	{
		pk := pr.PublicKey()

		b, err := rlp.EncodeToBytes(pk)
		t.NoError(err)
		t.NotEmpty(b)

		var key StellarPublicKey
		err = rlp.DecodeBytes(b, &key)
		t.NoError(err)
		t.NotEmpty(key)

		t.Equal(PublicKeyKind, key.Kind())
		t.True(pk.Equal(key))
	}
}

func (t *testStellarKeypair) TestEncodeRLPPublicKey() {
	st, _ := Stellar{}.New()
	pk := st.PublicKey()

	b, err := rlp.EncodeToBytes(pk)
	t.NoError(err)
	t.NotEmpty(b)

	var upk StellarPublicKey
	err = rlp.DecodeBytes(b, &upk)
	t.NoError(err)

	var upr StellarPrivateKey
	err = rlp.DecodeBytes(b, &upr)
	t.True(xerrors.Is(err, FailedToEncodeKeypairError))
	t.Contains(err.Error(), "not private")
}

func (t *testStellarKeypair) TestEncodeRLPPrivateKey() {
	st := Stellar{}
	pr, _ := st.New()

	b, err := rlp.EncodeToBytes(pr)
	t.NoError(err)
	t.NotEmpty(b)

	var upr StellarPrivateKey
	err = rlp.DecodeBytes(b, &upr)
	t.NoError(err)

	var upk StellarPublicKey
	err = rlp.DecodeBytes(b, &upk)
	t.True(xerrors.Is(err, FailedToEncodeKeypairError))
	t.Contains(err.Error(), "not public")
}

func (t *testStellarKeypair) TestSigning() {
	st := Stellar{}
	pr, _ := st.New()

	input := []byte("source")
	sig, err := pr.Sign(input)
	t.NoError(err)
	t.NotEmpty(sig)

	{ // valid input
		err = pr.PublicKey().Verify(input, sig)
		t.NoError(err)
	}

	{ // invalid input
		err = pr.PublicKey().Verify([]byte("killme"), sig)
		t.True(xerrors.Is(err, SignatureVerificationFailedError))
	}
}

func (t *testStellarKeypair) TestFromSeed() {
	seed := []byte("find me")

	pr0, _ := Stellar{}.NewFromSeed(seed)
	pr1, _ := Stellar{}.NewFromSeed(seed)

	t.True(pr0.Equal(pr1))
	t.True(pr0.PublicKey().Equal(pr1.PublicKey()))
}

func TestStellarKeypair(t *testing.T) {
	suite.Run(t, new(testStellarKeypair))
}
