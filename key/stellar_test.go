package key

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
)

type testStellarKeypair struct {
	suite.Suite
}

func (t *testStellarKeypair) SetupTest() {
	_ = hint.RegisterType(StellarPrivatekey{}.Hint().Type(), "stellar-privatekey")
	_ = hint.RegisterType(StellarPublickey{}.Hint().Type(), "stellar-publickey")
}

func (t *testStellarKeypair) TestNew() {
	kp, err := NewStellarPrivatekey()
	t.NoError(err)

	t.Implements((*Privatekey)(nil), kp)
}

func (t *testStellarKeypair) TestKeypairIsValid() {
	kp, _ := NewStellarPrivatekey()
	t.NoError(kp.IsValid())

	// empty Keypair
	empty := StellarPrivatekey{}
	t.True(xerrors.Is(empty.IsValid(), InvalidKeyError))
}

func (t *testStellarKeypair) TestKeypairExportKeys() {
	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)

	t.Equal("GAVAONBETT4MVPV2IYN2T7OB7ZTYXGNN4BFGZHUYBUYR6G4ACHZMDOQ6", kp.Publickey().String())
	t.Equal(seed, kp.String())
}

func (t *testStellarKeypair) TestPublickey() {
	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)

	t.Equal("GAVAONBETT4MVPV2IYN2T7OB7ZTYXGNN4BFGZHUYBUYR6G4ACHZMDOQ6", kp.Publickey().String())

	t.NoError(kp.IsValid())

	pk, err := NewStellarPublickeyFromString(kp.Publickey().String())
	t.NoError(err)

	t.True(kp.Publickey().Equal(pk))
}

func (t *testStellarKeypair) TestPublickeyEqual() {
	kp, _ := NewStellarPrivatekey()

	t.True(kp.Publickey().Equal(kp.Publickey()))

	nkp, _ := NewStellarPrivatekey()
	t.False(kp.Publickey().Equal(nkp.Publickey()))
}

func (t *testStellarKeypair) TestPrivatekey() {
	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)

	t.Equal(seed, kp.String())

	t.NoError(kp.IsValid())

	pk, err := NewStellarPrivatekeyFromString(kp.String())
	t.NoError(err)

	t.True(kp.Equal(pk))
}

func (t *testStellarKeypair) TestPrivatekeyEqual() {
	kp, _ := NewStellarPrivatekey()

	t.True(kp.Equal(kp))

	nkp, _ := NewStellarPrivatekey()
	t.False(kp.Equal(nkp))
}

func (t *testStellarKeypair) TestSign() {
	kp, _ := NewStellarPrivatekey()

	input := []byte("makeme")

	// sign
	sig, err := kp.Sign(input)
	t.NoError(err)
	t.NotNil(sig)

	// verify
	err = kp.Publickey().Verify(input, sig)
	t.NoError(err)
}

func TestStellarKeypair(t *testing.T) {
	suite.Run(t, new(testStellarKeypair))
}
