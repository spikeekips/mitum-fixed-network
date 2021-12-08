package key

import (
	"errors"
	"testing"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/stretchr/testify/suite"
)

type testBasePublickey struct {
	suite.Suite
}

func (t *testBasePublickey) TestNew() {
	priv := NewBasePrivatekey()
	t.NoError(priv.IsValid(nil))

	pub := priv.Publickey()

	t.Implements((*Publickey)(nil), pub)
}

func (t *testBasePublickey) TestParseBasePublickey() {
	priv := NewBasePrivatekey()
	pub := priv.Publickey()

	parsed, err := ParseBasePublickey(pub.String())
	t.NoError(err)

	t.True(pub.Equal(parsed))
}

func (t *testBasePublickey) TestInvalid() {
	priv := NewBasePrivatekey()
	pub := priv.Publickey().(BasePublickey)

	{ // empty *btcec.PublicKey
		n := pub
		n.k = nil
		err := n.IsValid(nil)
		t.True(errors.Is(err, InvalidKeyError))
		t.Contains(err.Error(), "empty btc PublicKey")
	}

	{ // empty *btcec.PublicKey
		n := pub
		n.s = ""
		err := n.IsValid(nil)
		t.True(errors.Is(err, InvalidKeyError))
		t.Contains(err.Error(), "empty publickey string")
	}

	{ // empty *btcec.PublicKey
		n := pub
		n.b = nil
		err := n.IsValid(nil)
		t.True(errors.Is(err, InvalidKeyError))
		t.Contains(err.Error(), "empty publickey []byte")
	}
}

func (t *testBasePublickey) TestEqual() {
	priv := NewBasePrivatekey()
	pub := priv.Publickey()

	privb := NewBasePrivatekey()
	pubb := privb.Publickey()

	t.True(pub.Equal(pub))
	t.False(pub.Equal(pubb))
	t.True(pubb.Equal(pubb))
	t.False(pub.Equal(nil))
	t.False(pubb.Equal(nil))
	t.False(pub.Equal(wrongHintedKey{Key: pub, ht: hint.NewHint(hint.Type("wrong"), "v0.0.1")}))
	t.True(pub.Equal(wrongHintedKey{Key: pub, ht: hint.NewHint(BasePublickeyType, "v0.0.1")}))
}

func (t *testBasePublickey) TestSign() {
	priv := NewBasePrivatekey()

	input := []byte("makeme")

	sig, err := priv.Sign(input)
	t.NoError(err)
	t.NotNil(sig)

	t.NoError(priv.Publickey().Verify(input, sig))

	{ // different input
		err = priv.Publickey().Verify([]byte("findme"), sig)
		t.Error(err)
		t.True(errors.Is(err, SignatureVerificationFailedError))
	}

	{ // wrong signature
		sig, err := priv.Sign([]byte("findme"))
		t.NoError(err)
		t.NotNil(sig)

		err = priv.Publickey().Verify(input, sig)
		t.Error(err)
		t.True(errors.Is(err, SignatureVerificationFailedError))
	}

	{ // different pubickey
		err = NewBasePrivatekey().Publickey().Verify(input, sig)
		t.Error(err)
		t.True(errors.Is(err, SignatureVerificationFailedError))
	}
}

func TestBasePublickey(t *testing.T) {
	suite.Run(t, new(testBasePublickey))
}

func testBasePublickeyEncode() *baseTestKeyEncode {
	s := new(baseTestKeyEncode)
	s.compare = func(a, b Key) {
		_, ok := a.(Publickey)
		s.True(ok)
		_, ok = b.(Publickey)
		s.True(ok)

		_, ok = a.(BasePublickey)
		s.True(ok)
		_, ok = b.(BasePublickey)
		s.True(ok)
	}

	return s
}

func TestBasePublickeJSON(t *testing.T) {
	s := testBasePublickeyEncode()
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		k := NewBasePrivatekey().Publickey()
		b, err := s.enc.Marshal(k)
		s.NoError(err)

		return k, b
	}
	s.decode = func(b []byte) Key {
		var uk BasePublickey
		s.NoError(s.enc.Unmarshal(b, &uk))

		return uk
	}

	suite.Run(t, s)
}

func TestBasePublickeyBSON(t *testing.T) {
	s := testBasePublickeyEncode()
	s.enc = bsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		k := NewBasePrivatekey().Publickey()
		b, err := s.enc.Marshal(struct {
			K Publickey
		}{K: k})
		s.NoError(err)

		return k, b
	}
	s.decode = func(b []byte) Key {
		var d struct {
			K BasePublickey
		}
		s.NoError(s.enc.Unmarshal(b, &d))

		return d.K
	}

	suite.Run(t, s)
}

func TestBasePublickeyDecoderJSON(t *testing.T) {
	s := testBasePublickeyEncode()
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		k := NewBasePrivatekey().Publickey()
		b, err := s.enc.Marshal(k)
		s.NoError(err)

		return k, b
	}
	s.decode = func(b []byte) Key {
		var d PublickeyDecoder
		s.NoError(s.enc.Unmarshal(b, &d))
		uk, err := d.Encode(s.enc)
		s.NoError(err)

		return uk
	}

	suite.Run(t, s)
}

func TestBasePublickeyDecoderBSON(t *testing.T) {
	s := testBasePublickeyEncode()
	s.enc = bsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		k := NewBasePrivatekey().Publickey()
		b, err := s.enc.Marshal(struct {
			K Publickey
		}{K: k})
		s.NoError(err)

		return k, b
	}
	s.decode = func(b []byte) Key {
		var d struct {
			K PublickeyDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &d))
		uk, err := d.K.Encode(s.enc)
		s.NoError(err)

		return uk
	}

	suite.Run(t, s)
}

func TestNilBasePublickeyDecoderJSON(t *testing.T) {
	s := testBasePublickeyEncode()
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		b, err := s.enc.Marshal(nil)
		s.NoError(err)

		return nil, b
	}
	s.decode = func(b []byte) Key {
		var d PublickeyDecoder
		s.NoError(s.enc.Unmarshal(b, &d))
		uk, err := d.Encode(s.enc)
		s.NoError(err)

		return uk
	}
	s.compare = func(a, b Key) {
		s.Nil(a)
		s.Nil(b)
	}

	suite.Run(t, s)
}

func TestNilBasePublickeyDecoderBSON(t *testing.T) {
	s := testBasePublickeyEncode()
	s.enc = bsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		b, err := s.enc.Marshal(struct {
			K Publickey
		}{})
		s.NoError(err)

		return nil, b
	}
	s.decode = func(b []byte) Key {
		var d struct {
			K PublickeyDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &d))
		uk, err := d.K.Encode(s.enc)
		s.NoError(err)

		return uk
	}
	s.compare = func(a, b Key) {
		s.Nil(a)
		s.Nil(b)
	}

	suite.Run(t, s)
}
