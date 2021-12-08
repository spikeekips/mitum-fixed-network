package key

import (
	"errors"
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/stretchr/testify/suite"
)

type testBasePrivatekey struct {
	suite.Suite
}

func (t *testBasePrivatekey) TestNew() {
	priv := NewBasePrivatekey()

	t.NoError(priv.IsValid(nil))

	t.Implements((*Privatekey)(nil), priv)
}

func (t *testBasePrivatekey) TestFromSeedStatic() {
	seed := "L1bQZCcDZKy342x8xjK9Hk935Nttm2jkApVVS2mn4Nqyxvu7nyGC"
	priv, err := NewBasePrivatekeyFromSeed(seed)
	t.NoError(err)

	t.Equal("KzBYiN3Qr1JuYNf7Eyc67PAC5bzBazopzwAQDVZj4jmya7sWTbCDmpr", priv.String())
	t.Equal("oxkQTcfKzrC67GE8ChZmZw8SBBBYefMp5859R2AZ8bB9mpu", priv.Publickey().String())
}

func (t *testBasePrivatekey) TestConflicts() {
	created := map[string]struct{}{}

	for i := 0; i < 400; i++ {
		if i%200 == 0 {
			t.T().Log("generated:", i)
		}

		priv := NewBasePrivatekey()
		upriv, err := ParseBasePrivatekey(priv.String())
		t.NoError(err)
		t.True(priv.Equal(upriv))

		upub, err := ParseBasePublickey(priv.Publickey().String())
		t.NoError(err)
		t.True(priv.Publickey().Equal(upub))

		_, found := created[priv.String()]
		t.False(found)

		if found {
			break
		}

		created[priv.String()] = struct{}{}
	}
}

func (t *testBasePrivatekey) TestConflictsSeed() {
	created := map[string]struct{}{}

	for i := 0; i < 400; i++ {
		if i%200 == 0 {
			t.T().Log("generated:", i)
		}

		priv, err := NewBasePrivatekeyFromSeed(util.UUID().String())
		t.NoError(err)

		upriv, err := ParseBasePrivatekey(priv.String())
		t.NoError(err)
		t.True(priv.Equal(upriv))

		upub, err := ParseBasePublickey(priv.Publickey().String())
		t.NoError(err)
		t.True(priv.Publickey().Equal(upub))

		_, found := created[priv.String()]
		t.False(found)

		if found {
			break
		}

		created[priv.String()] = struct{}{}
	}
}

func (t *testBasePrivatekey) TestFromSeedButTooShort() {
	seed := util.UUID().String()[:MinSeedSize-1]

	_, err := NewBasePrivatekeyFromSeed(seed)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "too short")
}

func (t *testBasePrivatekey) TestParseBasePrivatekey() {
	priv := NewBasePrivatekey()
	parsed, err := ParseBasePrivatekey(priv.String())
	t.NoError(err)

	t.True(priv.Equal(parsed))
}

func (t *testBasePrivatekey) TestParseBasePrivatekeyButEmpty() {
	_, err := ParseBasePrivatekey("")
	t.True(errors.Is(err, InvalidKeyError))
	t.Contains(err.Error(), "unknown privatekey string")

	_, err = ParseBasePrivatekey(string(BasePrivatekeyType))
	t.True(errors.Is(err, InvalidKeyError))
	t.Contains(err.Error(), "invalid privatekey string")

	_, err = ParseBasePrivatekey(util.UUID().String() + string(BasePrivatekeyType))
	t.True(errors.Is(err, InvalidKeyError))
	t.Contains(err.Error(), "malformed private key")
}

func (t *testBasePrivatekey) TestFromSeed() {
	seed := util.UUID().String() + util.UUID().String()

	priva, err := NewBasePrivatekeyFromSeed(seed)
	t.NoError(err)

	for i := 0; i < 400; i++ {
		if i%200 == 0 {
			t.T().Log("generated:", i)
		}

		b, err := NewBasePrivatekeyFromSeed(seed)
		t.NoError(err)

		t.True(priva.Equal(b))
	}
}

type wrongHintedKey struct {
	Key
	ht hint.Hint
}

func (k wrongHintedKey) Hint() hint.Hint {
	return k.ht
}

func (t *testBasePrivatekey) TestEqual() {
	priv := NewBasePrivatekey()
	b := NewBasePrivatekey()

	t.True(priv.Equal(priv))
	t.False(priv.Equal(b))
	t.True(b.Equal(b))
	t.False(priv.Equal(nil))
	t.False(b.Equal(nil))
	t.False(priv.Equal(wrongHintedKey{Key: priv, ht: hint.NewHint(hint.Type("wrong"), "v0.0.1")}))
	t.True(priv.Equal(wrongHintedKey{Key: priv, ht: hint.NewHint(BasePrivatekeyType, "v0.0.1")}))
}

func (t *testBasePrivatekey) TestInvalid() {
	{ // nil wif
		priv := NewBasePrivatekey()
		t.NoError(priv.IsValid(nil))

		priv.wif = nil
		err := priv.IsValid(nil)
		t.True(errors.Is(err, InvalidKeyError))
		t.True(errors.Is(err, isvalid.InvalidError))
		t.Contains(err.Error(), "empty btc wif")
	}

	{ // nil wif.PrivKey
		priv := NewBasePrivatekey()
		t.NoError(priv.IsValid(nil))

		priv.wif.PrivKey = nil
		err := priv.IsValid(nil)
		t.True(errors.Is(err, InvalidKeyError))
		t.Contains(err.Error(), "empty btc wif.PrivKey")
	}

	{ // empty string
		priv := NewBasePrivatekey()
		t.NoError(priv.IsValid(nil))

		priv.s = ""
		err := priv.IsValid(nil)
		t.True(errors.Is(err, InvalidKeyError))
		t.Contains(err.Error(), "empty privatekey string")
	}

	{ // empty [byte
		priv := NewBasePrivatekey()
		t.NoError(priv.IsValid(nil))

		priv.b = nil
		err := priv.IsValid(nil)
		t.True(errors.Is(err, InvalidKeyError))
		t.Contains(err.Error(), "empty privatekey []byte")
	}
}

func TestBasePrivatekey(t *testing.T) {
	suite.Run(t, new(testBasePrivatekey))
}

type baseTestKeyEncode struct {
	suite.Suite
	enc     encoder.Encoder
	encode  func() (Key, []byte)
	decode  func([]byte) Key
	compare func(Key, Key)
}

func (t *baseTestKeyEncode) SetupSuite() {
	t.enc.Add(BasePrivatekey{})
	t.enc.Add(BasePublickey{})
}

func (t *baseTestKeyEncode) TestDecode() {
	k, b := t.encode()

	uk := t.decode(b)

	if k != nil && uk != nil {
		t.True(k.Hint().Equal(uk.Hint()))
		t.Equal(k.String(), uk.String())
	}

	t.compare(k, uk)
}

func testBasePrivatekeyEncode() *baseTestKeyEncode {
	s := new(baseTestKeyEncode)
	s.compare = func(a, b Key) {
		_, ok := a.(Privatekey)
		s.True(ok)
		_, ok = b.(Privatekey)
		s.True(ok)

		_, ok = a.(BasePrivatekey)
		s.True(ok)
		_, ok = b.(BasePrivatekey)
		s.True(ok)
	}

	return s
}

func TestBasePrivatekeyJSON(t *testing.T) {
	s := testBasePrivatekeyEncode()
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		k := NewBasePrivatekey()
		b, err := s.enc.Marshal(k)
		s.NoError(err)

		return k, b
	}
	s.decode = func(b []byte) Key {
		var uk BasePrivatekey
		s.NoError(s.enc.Unmarshal(b, &uk))

		return uk
	}

	suite.Run(t, s)
}

func TestBasePrivatekeyBSON(t *testing.T) {
	s := testBasePrivatekeyEncode()
	s.enc = bsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		k := NewBasePrivatekey()
		b, err := s.enc.Marshal(struct {
			K BasePrivatekey
		}{K: k})
		s.NoError(err)

		return k, b
	}
	s.decode = func(b []byte) Key {
		var d struct {
			K BasePrivatekey
		}
		s.NoError(s.enc.Unmarshal(b, &d))

		return d.K
	}

	suite.Run(t, s)
}

func TestBasePrivatekeyDecoderJSON(t *testing.T) {
	s := testBasePrivatekeyEncode()
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		k := NewBasePrivatekey()
		b, err := s.enc.Marshal(k)
		s.NoError(err)

		return k, b
	}
	s.decode = func(b []byte) Key {
		var d PrivatekeyDecoder
		s.NoError(s.enc.Unmarshal(b, &d))
		uk, err := d.Encode(s.enc)
		s.NoError(err)

		return uk
	}

	suite.Run(t, s)
}

func TestBasePrivatekeyDecoderBSON(t *testing.T) {
	s := testBasePrivatekeyEncode()
	s.enc = bsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		k := NewBasePrivatekey()
		b, err := s.enc.Marshal(struct {
			K BasePrivatekey
		}{K: k})
		s.NoError(err)

		return k, b
	}
	s.decode = func(b []byte) Key {
		var d struct {
			K PrivatekeyDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &d))
		uk, err := d.K.Encode(s.enc)
		s.NoError(err)

		return uk
	}

	suite.Run(t, s)
}

func TestNilBasePrivatekeyDecoderJSON(t *testing.T) {
	s := testBasePrivatekeyEncode()
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		b, err := s.enc.Marshal(nil)
		s.NoError(err)

		return nil, b
	}
	s.decode = func(b []byte) Key {
		var d PrivatekeyDecoder
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

func TestNilBasePrivatekeyDecoderBSON(t *testing.T) {
	s := testBasePrivatekeyEncode()
	s.enc = bsonenc.NewEncoder()
	s.encode = func() (Key, []byte) {
		b, err := s.enc.Marshal(struct {
			K Privatekey
		}{})
		s.NoError(err)

		return nil, b
	}
	s.decode = func(b []byte) Key {
		var d struct {
			K PrivatekeyDecoder
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
