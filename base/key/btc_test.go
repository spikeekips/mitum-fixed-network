package key

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testBTCKey struct {
	suite.Suite
}

func (t *testBTCKey) TestNew() {
	kp, err := NewBTCPrivatekey()
	t.NoError(err)

	t.Implements((*Privatekey)(nil), kp)
}

func (t *testBTCKey) TestKeypairIsValid() {
	kp, _ := NewBTCPrivatekey()
	t.NoError(kp.IsValid(nil))

	// empty Keypair
	empty := BTCPrivatekey{}
	t.True(xerrors.Is(empty.IsValid(nil), InvalidKeyError))
}

func (t *testBTCKey) TestKeypairExportKeys() {
	priv := "L1bQZCcDZKy342x8xjK9Hk935Nttm2jkApVVS2mn4Nqyxvu7nyGC"
	kp, _ := NewBTCPrivatekeyFromString(priv)

	t.Equal("27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1", kp.Publickey().String())
}

func (t *testBTCKey) TestPublickey() {
	priv := "L1bQZCcDZKy342x8xjK9Hk935Nttm2jkApVVS2mn4Nqyxvu7nyGC"
	kp, _ := NewBTCPrivatekeyFromString(priv)

	t.Equal("27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb-0113:0.0.1", kp.Publickey().String())

	t.NoError(kp.IsValid(nil))

	_, s, err := hint.ParseHintedString(kp.Publickey().String())
	t.NoError(err)

	ukp, err := NewBTCPublickeyFromString(s)
	t.NoError(err)

	t.True(kp.Publickey().Equal(ukp))
}

func (t *testBTCKey) TestPublickeyEqual() {
	kp, _ := NewBTCPrivatekey()

	t.True(kp.Publickey().Equal(kp.Publickey()))

	nkp, _ := NewBTCPrivatekey()
	t.False(kp.Publickey().Equal(nkp.Publickey()))
}

func (t *testBTCKey) TestPrivatekey() {
	priv := "L1bQZCcDZKy342x8xjK9Hk935Nttm2jkApVVS2mn4Nqyxvu7nyGC"
	kp, _ := NewBTCPrivatekeyFromString(priv)

	t.NoError(kp.IsValid(nil))

	_, s, err := hint.ParseHintedString(kp.String())
	t.NoError(err)

	ukp, _ := NewBTCPrivatekeyFromString(s)
	t.True(kp.Equal(ukp))
}

func (t *testBTCKey) TestPrivatekeyEqual() {
	kp, _ := NewBTCPrivatekey()

	t.True(kp.Equal(kp))

	nkp, _ := NewBTCPrivatekey()
	t.False(kp.Equal(nkp))
}

func (t *testBTCKey) TestSign() {
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

func (t *testBTCKey) TestSignInvalidInput() {
	kp, _ := NewBTCPrivatekey()

	b := []byte(util.UUID().String())

	input := b
	input = append(input, []byte("findme000")...)

	sig, err := kp.Sign(input)
	t.NoError(err)
	t.NotNil(sig)

	{
		err = kp.Publickey().Verify(input, sig)
		t.NoError(err)
	}

	{
		newInput := b
		newInput = append(newInput, []byte("showme")...)

		err = kp.Publickey().Verify(newInput, sig)
		t.True(xerrors.Is(err, SignatureVerificationFailedError))
	}
}

func TestBTCKey(t *testing.T) {
	suite.Run(t, new(testBTCKey))
}
