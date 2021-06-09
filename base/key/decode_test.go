package key

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type baseTestHintedString struct {
	suite.Suite
	encs   *encoder.Encoders
	je     *jsonenc.Encoder
	be     *bsonenc.Encoder
	newKey func() (Key, error)
}

func (t *baseTestHintedString) SetupSuite() {
	t.je = jsonenc.NewEncoder()
	t.be = bsonenc.NewEncoder()

	t.encs = encoder.NewEncoders()
	t.encs.AddEncoder(t.je)
	t.encs.AddEncoder(t.be)

	k, _ := t.newKey()
	t.encs.TestAddHinter(k)
}

func (t *baseTestHintedString) TestDecodeBSON() {
	kp, err := t.newKey()
	t.NoError(err)

	mkp := struct {
		KP Key
	}{
		KP: kp,
	}

	b, err := bsonenc.Marshal(mkp)
	t.NoError(err)

	var ukp Key
	switch kp.(type) {
	case Privatekey:
		i := struct {
			KP *PrivatekeyDecoder
		}{}
		t.NoError(bsonenc.Unmarshal(b, &i))

		k, err := i.KP.Encode(t.be)
		t.NoError(err)

		ukp = k
	case Publickey:
		i := struct {
			KP *PublickeyDecoder
		}{}
		t.NoError(bsonenc.Unmarshal(b, &i))

		k, err := i.KP.Encode(t.be)
		t.NoError(err)

		ukp = k
	}

	t.Equal(kp.String(), ukp.String())
	t.True(kp.Equal(ukp))
}

func (t *baseTestHintedString) TestDecodeJSON() {
	kp, err := t.newKey()
	t.NoError(err)

	mkp := struct {
		KP Key
	}{
		KP: kp,
	}

	b, err := jsonenc.Marshal(mkp)
	t.NoError(err)

	var ukp Key
	switch kp.(type) {
	case Privatekey:
		i := struct {
			KP *PrivatekeyDecoder
		}{}
		t.NoError(jsonenc.Unmarshal(b, &i))

		k, err := i.KP.Encode(t.je)
		t.NoError(err)

		ukp = k
	case Publickey:
		i := struct {
			KP *PublickeyDecoder
		}{}
		t.NoError(jsonenc.Unmarshal(b, &i))

		k, err := i.KP.Encode(t.je)
		t.NoError(err)

		ukp = k
	}

	t.True(kp.Equal(ukp))
	t.Equal(kp.String(), ukp.String())
}

type testHintedString struct {
	baseTestHintedString
}

func TestHintedStringEtherPrivatekey(t *testing.T) {
	s := new(testHintedString)
	s.newKey = func() (Key, error) {
		return NewEtherPrivatekey()
	}

	suite.Run(t, s)
}

func TestHintedStringEtherPublickey(t *testing.T) {
	s := new(testHintedString)
	s.newKey = func() (Key, error) {
		k, _ := NewEtherPrivatekey()

		return k.Publickey(), nil
	}

	suite.Run(t, s)
}

func TestHintedStringBTCPrivatekey(t *testing.T) {
	s := new(testHintedString)
	s.newKey = func() (Key, error) {
		return NewBTCPrivatekey()
	}

	suite.Run(t, s)
}

func TestHintedStringBTCPublickey(t *testing.T) {
	s := new(testHintedString)
	s.newKey = func() (Key, error) {
		k, _ := NewBTCPrivatekey()

		return k.Publickey(), nil
	}

	suite.Run(t, s)
}

func TestHintedStringStellarPrivatekey(t *testing.T) {
	s := new(testHintedString)
	s.newKey = func() (Key, error) {
		return NewStellarPrivatekey()
	}

	suite.Run(t, s)
}

func TestHintedStringStellarPublickey(t *testing.T) {
	s := new(testHintedString)
	s.newKey = func() (Key, error) {
		k, _ := NewStellarPrivatekey()

		return k.Publickey(), nil
	}

	suite.Run(t, s)
}
