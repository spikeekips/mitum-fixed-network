package key

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
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
	empty := BTCPrivatekey{BaseKey: NewBaseKey(BTCPrivatekeyHint, nil)}
	t.True(errors.Is(empty.IsValid(nil), InvalidKeyError))
}

func (t *testBTCKey) TestKeypairExportKeys() {
	priv := "L1bQZCcDZKy342x8xjK9Hk935Nttm2jkApVVS2mn4Nqyxvu7nyGC"
	kp, _ := NewBTCPrivatekeyFromString(priv)

	t.Equal(hint.NewHintedString(BTCPublickeyHint, "27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb").String(), kp.Publickey().String())
}

func (t *testBTCKey) TestPublickey() {
	priv := "L1bQZCcDZKy342x8xjK9Hk935Nttm2jkApVVS2mn4Nqyxvu7nyGC"
	kp, _ := NewBTCPrivatekeyFromString(priv)

	t.Equal(hint.NewHintedString(BTCPublickeyHint, "27phogA4gmbMGfg321EHfx5eABkL7KAYuDPRGFoyQtAUb").String(), kp.Publickey().String())

	t.NoError(kp.IsValid(nil))

	ukp, err := NewBTCPublickeyFromString(kp.Publickey().Raw())
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

	ukp, _ := NewBTCPrivatekeyFromString(priv)
	t.True(kp.Equal(ukp))

	t.Equal(priv, kp.Raw())
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
		t.True(errors.Is(err, SignatureVerificationFailedError))
	}
}

func TestBTCKey(t *testing.T) {
	suite.Run(t, new(testBTCKey))
}
