package key

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
)

type testBTCKeypair struct {
	suite.Suite
}

func (t *testBTCKeypair) SetupTest() {
	_ = hint.RegisterType(BTCPrivatekey{}.Hint().Type(), "btc-privatekey")
	_ = hint.RegisterType(BTCPublickey{}.Hint().Type(), "btc-publickey")
}

func (t *testBTCKeypair) TestNew() {
	kp, err := NewBTCPrivatekey()
	t.NoError(err)

	t.Implements((*Privatekey)(nil), kp)
}

func (t *testBTCKeypair) TestKeypairIsValid() {
	kp, _ := NewBTCPrivatekey()
	t.NoError(kp.IsValid())

	// empty Keypair
	empty := BTCPrivatekey{}
	t.True(xerrors.Is(empty.IsValid(), InvalidKeyError))
}

func (t *testBTCKeypair) TestKeypairExportKeys() {
	priv := "L1bQZCcDZKy342x8xjK9Hk935Nttm2jkApVVS2mn4Nqyxvu7nyGC"
	kp, _ := NewBTCPrivatekeyFromString(priv)

	t.Equal("27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb", kp.Publickey().String())
	t.Equal(priv, kp.String())
}

func (t *testBTCKeypair) TestPublickey() {
	priv := "L1bQZCcDZKy342x8xjK9Hk935Nttm2jkApVVS2mn4Nqyxvu7nyGC"
	kp, _ := NewBTCPrivatekeyFromString(priv)

	t.Equal("27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb", kp.Publickey().String())

	t.NoError(kp.IsValid())

	pk, err := NewBTCPublickeyFromString(kp.Publickey().String())
	t.NoError(err)

	t.True(kp.Publickey().Equal(pk))
}

func (t *testBTCKeypair) TestPublickeyEqual() {
	kp, _ := NewBTCPrivatekey()

	t.True(kp.Publickey().Equal(kp.Publickey()))

	nkp, _ := NewBTCPrivatekey()
	t.False(kp.Publickey().Equal(nkp.Publickey()))
}

func (t *testBTCKeypair) TestPrivatekey() {
	priv := "L1bQZCcDZKy342x8xjK9Hk935Nttm2jkApVVS2mn4Nqyxvu7nyGC"
	kp, _ := NewBTCPrivatekeyFromString(priv)

	t.Equal(priv, kp.String())

	t.NoError(kp.IsValid())

	pk, err := NewBTCPrivatekeyFromString(kp.String())
	t.NoError(err)

	t.True(kp.Equal(pk))
}

func (t *testBTCKeypair) TestPrivatekeyEqual() {
	kp, _ := NewBTCPrivatekey()

	t.True(kp.Equal(kp))

	nkp, _ := NewBTCPrivatekey()
	t.False(kp.Equal(nkp))
}

func (t *testBTCKeypair) TestSign() {
	kp, _ := NewBTCPrivatekey()

	input := []byte("makeme")

	// sign
	sig, err := kp.Sign(input)
	t.NoError(err)
	t.NotNil(sig)

	// verify
	err = kp.Publickey().Verify(input, sig)
	t.NoError(err)
}

func TestBTCKeypair(t *testing.T) {
	suite.Run(t, new(testBTCKeypair))
}
