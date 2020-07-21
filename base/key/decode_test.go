package key

import (
	"encoding/json"
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
	t.encs.AddHinter(k)
}

func (t *baseTestHintedString) TestDecodeJSON() {
	kp, err := t.newKey()
	t.NoError(err)

	b, err := json.Marshal(kp)
	t.NoError(err)

	var kd encoder.HintedString
	t.NoError(json.Unmarshal(b, &kd))

	t.NoError(kd.IsValid(nil))

	ukp, err := kd.Encode(t.je)
	t.NoError(err)

	t.True(kp.Equal(ukp.(Key)))
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

	var ukd struct {
		KP encoder.HintedString
	}

	t.NoError(bsonenc.Unmarshal(b, &ukd))

	kd := ukd.KP
	t.NoError(kd.IsValid(nil))

	ukp, err := kd.Encode(t.be)
	t.NoError(err)

	t.True(kp.Equal(ukp.(Key)))
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
