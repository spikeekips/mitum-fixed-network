package key

import (
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testStellarKey struct {
	suite.Suite
}

func (t *testStellarKey) TestNew() {
	kp, err := NewStellarPrivatekey()
	t.NoError(err)

	t.Implements((*Privatekey)(nil), kp)
}

func (t *testStellarKey) TestKeypairIsValid() {
	kp, _ := NewStellarPrivatekey()
	t.NoError(kp.IsValid(nil))

	// empty Keypair
	empty := StellarPrivatekey{}
	t.True(xerrors.Is(empty.IsValid(nil), InvalidKeyError))
}

func (t *testStellarKey) TestKeypairExportKeys() {
	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)

	t.Equal(hint.NewHintedString(StellarPublickeyHint, "GAVAONBETT4MVPV2IYN2T7OB7ZTYXGNN4BFGZHUYBUYR6G4ACHZMDOQ6").String(), kp.Publickey().String())
	t.Equal(hint.NewHintedString(StellarPrivatekeyHint, "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673").String(), kp.String())
}

func (t *testStellarKey) TestPublickey() {
	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)

	t.Equal(hint.NewHintedString(StellarPublickeyHint, "GAVAONBETT4MVPV2IYN2T7OB7ZTYXGNN4BFGZHUYBUYR6G4ACHZMDOQ6").String(), kp.Publickey().String())

	t.NoError(kp.IsValid(nil))

	pk, err := NewStellarPublickeyFromString(kp.Publickey().Raw())
	t.NoError(err)

	t.True(kp.Publickey().Equal(pk))
}

func (t *testStellarKey) TestPublickeyEqual() {
	kp, _ := NewStellarPrivatekey()

	t.True(kp.Publickey().Equal(kp.Publickey()))

	nkp, _ := NewStellarPrivatekey()
	t.False(kp.Publickey().Equal(nkp.Publickey()))
}

func (t *testStellarKey) TestPrivatekey() {
	seed := "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673"
	kp, _ := NewStellarPrivatekeyFromString(seed)

	t.Equal(hint.NewHintedString(StellarPrivatekeyHint, "SCD6GQMWGDQT33QOCNKYKRJZL3YWFSLBVQSL6ICVWBUYQZCBFYUQY673").String(), kp.String())

	t.NoError(kp.IsValid(nil))

	pk, err := NewStellarPrivatekeyFromString(seed)
	t.NoError(err)

	t.True(kp.Equal(pk))
}

func (t *testStellarKey) TestPrivatekeyEqual() {
	kp, _ := NewStellarPrivatekey()

	t.True(kp.Equal(kp))

	nkp, _ := NewStellarPrivatekey()
	t.False(kp.Equal(nkp))
}

func (t *testStellarKey) TestSign() {
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

func (t *testStellarKey) TestSignInvalidInput() {
	kp, _ := NewStellarPrivatekey()

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

func TestStellarKey(t *testing.T) {
	suite.Run(t, new(testStellarKey))
}
