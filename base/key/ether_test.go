package key

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
)

type testEtherKey struct {
	suite.Suite
}

func (t *testEtherKey) TestNew() {
	kp, err := NewEtherPrivatekey()
	t.NoError(err)

	t.Implements((*Privatekey)(nil), kp)
}

func (t *testEtherKey) TestKeypairIsValid() {
	kp, _ := NewEtherPrivatekey()
	t.NoError(kp.IsValid(nil))

	// empty Keypair
	empty := EtherPrivatekey{BaseKey: NewBaseKey(EtherPrivatekeyHint, nil)}
	t.True(errors.Is(empty.IsValid(nil), InvalidKeyError))
}

func (t *testEtherKey) TestKeypairExportKeys() {
	priv := "1940008c14106a4d7124f984075ff4295adb325cca97caa4431cfb83f04aa7f2"
	kp, err := NewEtherPrivatekeyFromString(priv)
	t.NoError(err)

	t.Equal(hint.NewHintedString(EtherPublickeyHint, "04cd279abff49a644f77f001baa1aba98880368d5a5cf476eb79e2c375a386edf495544f201d1774fbce4c5ef11e2de9c83f423d662d9d69147fcc6d3f96e81a75").String(), kp.Publickey().String())
	t.Equal(hint.NewHintedString(EtherPrivatekeyHint, "1940008c14106a4d7124f984075ff4295adb325cca97caa4431cfb83f04aa7f2").String(), kp.String())
}

func (t *testEtherKey) TestPublickey() {
	priv := "1940008c14106a4d7124f984075ff4295adb325cca97caa4431cfb83f04aa7f2"
	kp, _ := NewEtherPrivatekeyFromString(priv)

	t.Equal(hint.NewHintedString(EtherPublickeyHint, "04cd279abff49a644f77f001baa1aba98880368d5a5cf476eb79e2c375a386edf495544f201d1774fbce4c5ef11e2de9c83f423d662d9d69147fcc6d3f96e81a75").String(), kp.Publickey().String())

	t.NoError(kp.IsValid(nil))

	pk, err := NewEtherPublickeyFromString(kp.Publickey().Raw())
	t.NoError(err)

	t.True(kp.Publickey().Equal(pk))
}

func (t *testEtherKey) TestPublickeyEqual() {
	kp, _ := NewEtherPrivatekey()

	t.True(kp.Publickey().Equal(kp.Publickey()))

	nkp, _ := NewEtherPrivatekey()
	t.False(kp.Publickey().Equal(nkp.Publickey()))
}

func (t *testEtherKey) TestPrivatekey() {
	priv := "1940008c14106a4d7124f984075ff4295adb325cca97caa4431cfb83f04aa7f2"
	kp, _ := NewEtherPrivatekeyFromString(priv)

	t.Equal(hint.NewHintedString(EtherPrivatekeyHint, "1940008c14106a4d7124f984075ff4295adb325cca97caa4431cfb83f04aa7f2").String(), kp.String())

	t.NoError(kp.IsValid(nil))

	pk, err := NewEtherPrivatekeyFromString(priv)
	t.NoError(err)

	t.True(kp.Equal(pk))

	t.Equal(priv, kp.Raw())
}

func (t *testEtherKey) TestPrivatekeyEqual() {
	kp, _ := NewEtherPrivatekey()

	t.True(kp.Equal(kp))

	nkp, _ := NewEtherPrivatekey()
	t.False(kp.Equal(nkp))
}

func (t *testEtherKey) TestSign() {
	kp, _ := NewEtherPrivatekey()

	input := []byte("makeme")

	// sign
	sig, err := kp.Sign(input)
	t.NoError(err)
	t.NotNil(sig)

	// verify
	err = kp.Publickey().Verify(input, sig)
	t.NoError(err)
}

func (t *testEtherKey) TestSignInvalidInput() {
	kp, _ := NewEtherPrivatekey()

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

func TestEtherKey(t *testing.T) {
	suite.Run(t, new(testEtherKey))
}
