package base

import (
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/stretchr/testify/suite"
)

type testStringAddress struct {
	suite.Suite
}

func (t *testStringAddress) TestNew() {
	ad := NewStringAddress("abc")
	err := ad.IsValid(nil)
	t.NoError(err)

	t.Implements((*Address)(nil), ad)
}

func (t *testStringAddress) TestEmpty() {
	{ // empty
		ad := NewStringAddress("")
		err := ad.IsValid(nil)
		t.Error(err)
		t.True(errors.Is(err, isvalid.InvalidError))
		t.Contains(err.Error(), "too short")
	}

	{ // short
		ad := NewStringAddress(strings.Repeat("a", MinAddressSize-AddressTypeSize-1))
		err := ad.IsValid(nil)
		t.Error(err)
		t.True(errors.Is(err, isvalid.InvalidError))
		t.Contains(err.Error(), "too short")
	}

	{ // long
		ad := NewStringAddress(strings.Repeat("a", MaxAddressSize) + "a")
		err := ad.IsValid(nil)
		t.Error(err)
		t.True(errors.Is(err, isvalid.InvalidError))
		t.Contains(err.Error(), "too long")
	}
}

func (t *testStringAddress) TestStringWithType() {
	ad := NewStringAddress("abc")
	err := ad.IsValid(nil)
	t.NoError(err)

	t.True(strings.HasSuffix(ad.String(), StringAddressType.String()))
}

func (t *testStringAddress) TestWrongHint() {
	ad := NewStringAddress("abc")
	ad.ht = hint.NewHint(hint.Type("aaa"), "v0.0.1")
	err := ad.IsValid(nil)
	t.NotNil(err)
	t.True(errors.Is(err, isvalid.InvalidError))
	t.Contains(err.Error(), "wrong type of string address")
}

func (t *testStringAddress) TestParse() {
	t.Run("valid", func() {
		ad := NewStringAddress("abc")
		t.NoError(ad.IsValid(nil))

		uad, err := ParseStringAddress(ad.String())
		t.NoError(err)

		t.True(ad.Equal(uad))
	})

	t.Run("wrong type", func() {
		ad := NewStringAddress("abc")
		t.NoError(ad.IsValid(nil))

		_, err := ParseStringAddress(ad.s[:len(ad.s)-AddressTypeSize] + "000")
		t.NotNil(err)
		t.True(errors.Is(err, isvalid.InvalidError))
		t.Contains(err.Error(), "wrong hint of string address")
	})
}

type wrongHintedAddress struct {
	Address
	ht hint.Hint
}

func (k wrongHintedAddress) Hint() hint.Hint {
	return k.ht
}

func (t *testStringAddress) TestEqual() {
	a := NewStringAddress("abc")
	b := NewStringAddress("abc")
	t.True(a.Equal(b))

	t.False(a.Equal(wrongHintedAddress{Address: a, ht: hint.NewHint(hint.Type("wrong"), "v0.0.1")}))
	t.True(a.Equal(wrongHintedAddress{Address: a, ht: hint.NewHint(StringAddressType, "v0.0.1")}))
}

func (t *testStringAddress) TestFormat() {
	uuidString := util.UUID().String()

	cases := []struct {
		name     string
		s        string
		expected string
		err      string
	}{
		{name: "uuid", s: uuidString, expected: uuidString + StringAddressType.String()},
		{name: "blank first", s: " showme", err: "has blank"},
		{name: "blank inside", s: "sh owme", err: "has blank"},
		{name: "blank inside #1", s: "sh\towme", err: "has blank"},
		{name: "blank ends", s: "showme ", err: "has blank"},
		{name: "blank ends, tab", s: "showme\t", err: "has blank"},
		{name: "has underscore", s: "showm_e", expected: "showm_e" + StringAddressType.String()},
		{name: "has plus sign", s: "showm+e", err: "invalid string address string"},
		{name: "has at sign", s: "showm@e", expected: "showm@e" + StringAddressType.String()},
		{name: "has dot", s: "showm.e", expected: "showm.e" + StringAddressType.String()},
		{name: "has dot #1", s: "showme.", err: "invalid string address string"},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				r := NewStringAddress(c.s)
				err := r.IsValid(nil)
				if err != nil {
					if len(c.err) < 1 {
						t.NoError(err, "%d: %v", i, c.name)
					} else {
						t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
					}
				} else if len(c.err) > 0 {
					t.NoError(errors.Errorf(c.err), "%d: %v", i, c.name)
				} else {
					t.Equal(c.expected, r.String(), "%d: %v; %v != %v", i, c.name, c.expected, r.String())
				}
			},
		)
	}
}

func TestStringAddress(t *testing.T) {
	suite.Run(t, new(testStringAddress))
}

type testStringAddressEncode struct {
	suite.Suite
	enc     encoder.Encoder
	encode  func() (Address, []byte)
	decode  func([]byte) (Address, error)
	compare func(Address, Address)
}

func (t *testStringAddressEncode) TestDecode() {
	ad, b := t.encode()
	t.enc.Add(ad)

	uad, err := t.decode(b)
	if err != nil {
		return
	}

	if t.compare != nil {
		t.compare(ad, uad)

		return
	}

	_, ok := ad.(StringAddress)
	t.True(ok)
	_, ok = uad.(StringAddress)
	t.True(ok)

	t.True(ad.Hint().Equal(uad.Hint()))
	t.True(ad.Equal(uad))
}

func TestStringAddressDecoderJSON(t *testing.T) {
	s := new(testStringAddressEncode)
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Address, []byte) {
		ad := NewStringAddress(util.UUID().String())
		b, err := s.enc.Marshal(struct {
			A StringAddress
		}{A: ad})
		s.NoError(err)

		return ad, b
	}
	s.decode = func(b []byte) (Address, error) {
		var u struct {
			A AddressDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &u))

		uad, err := u.A.Encode(s.enc)
		s.NoError(err)

		return uad, nil
	}

	suite.Run(t, s)
}

func TestStringAddressDecoderBSON(t *testing.T) {
	s := new(testStringAddressEncode)
	s.enc = bsonenc.NewEncoder()
	s.encode = func() (Address, []byte) {
		ad := NewStringAddress(util.UUID().String())
		b, err := s.enc.Marshal(struct {
			A StringAddress
		}{A: ad})
		s.NoError(err)

		return ad, b
	}
	s.decode = func(b []byte) (Address, error) {
		var u struct {
			A AddressDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &u))

		uad, err := u.A.Encode(s.enc)
		s.NoError(err)

		return uad, nil
	}

	suite.Run(t, s)
}

func TestHintedStringAddressDecoderJSON(t *testing.T) {
	nht := hint.NewHint(hint.Type("nht"), "v0.0.1")

	s := new(testStringAddressEncode)
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Address, []byte) {
		ad := NewStringAddressWithHint(nht, util.UUID().String())
		b, err := s.enc.Marshal(struct {
			A StringAddress
		}{A: ad})
		s.NoError(err)

		return ad, b
	}
	s.decode = func(b []byte) (Address, error) {
		var u struct {
			A AddressDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &u))

		uad, err := u.A.Encode(s.enc)
		s.NoError(err)

		return uad, nil
	}

	suite.Run(t, s)
}

func TestHintedStringAddressDecoderBSON(t *testing.T) {
	nht := hint.NewHint(hint.Type("nht"), "v0.0.1")

	s := new(testStringAddressEncode)
	s.enc = bsonenc.NewEncoder()
	s.encode = func() (Address, []byte) {
		ad := NewStringAddressWithHint(nht, util.UUID().String())
		b, err := s.enc.Marshal(struct {
			A StringAddress
		}{A: ad})
		s.NoError(err)

		return ad, b
	}
	s.decode = func(b []byte) (Address, error) {
		var u struct {
			A AddressDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &u))

		uad, err := u.A.Encode(s.enc)
		s.NoError(err)

		return uad, nil
	}

	suite.Run(t, s)
}

func TestWrongHintedStringAddressDecoderJSON(t *testing.T) {
	nht := hint.NewHint(hint.Type("wrong"), "v0.0.1")

	s := new(testStringAddressEncode)
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Address, []byte) {
		ad := NewStringAddressWithHint(nht, util.UUID().String())
		b, err := s.enc.Marshal(struct {
			A StringAddress
		}{A: ad})
		s.NoError(err)

		return ad, b
	}
	s.decode = func(b []byte) (Address, error) {
		var u struct {
			A AddressDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &u))

		_, err := u.A.Encode(s.enc)
		s.NotNil(err)
		s.Contains(err.Error(), "failed to find unpacker")

		return nil, err
	}

	suite.Run(t, s)
}

func TestInvalidHintedStringAddressDecoderJSON(t *testing.T) {
	nht := hint.NewHint(hint.Type("uht"), "v0.0.1")

	s := new(testStringAddressEncode)
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Address, []byte) {
		ad := NewStringAddressWithHint(nht, "s")
		b, err := s.enc.Marshal(struct {
			A StringAddress
		}{A: ad})
		s.NoError(err)

		return ad, b
	}
	s.decode = func(b []byte) (Address, error) {
		var u struct {
			A AddressDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &u))

		uad, err := u.A.Encode(s.enc)
		s.NoError(err)

		return uad, nil
	}
	s.compare = func(a, b Address) {
		s.NotEqual(a.String(), b.String())

		err := a.IsValid(nil)
		s.NotNil(err)
		s.Contains(err.Error(), "too short")

		err = b.IsValid(nil)
		s.NotNil(err)
		s.Contains(err.Error(), "too short")
	}

	suite.Run(t, s)
}

func TestInvalidHintedStringAddressDecoderBSON(t *testing.T) {
	nht := hint.NewHint(hint.Type("uht"), "v0.0.1")

	s := new(testStringAddressEncode)
	s.enc = bsonenc.NewEncoder()
	s.encode = func() (Address, []byte) {
		ad := NewStringAddressWithHint(nht, "s")
		b, err := s.enc.Marshal(struct {
			A StringAddress
		}{A: ad})
		s.NoError(err)

		return ad, b
	}
	s.decode = func(b []byte) (Address, error) {
		var u struct {
			A AddressDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &u))

		uad, err := u.A.Encode(s.enc)
		s.NoError(err)

		return uad, nil
	}
	s.compare = func(a, b Address) {
		s.NotEqual(a.String(), b.String())

		err := a.IsValid(nil)
		s.NotNil(err)
		s.Contains(err.Error(), "too short")

		err = b.IsValid(nil)
		s.NotNil(err)
		s.Contains(err.Error(), "too short")
	}

	suite.Run(t, s)
}

func TestNilStringAddressDecoderJSON(t *testing.T) {
	nht := hint.NewHint(hint.Type("uht"), "v0.0.1")

	s := new(testStringAddressEncode)
	s.enc = jsonenc.NewEncoder()
	s.encode = func() (Address, []byte) {
		ad := NewStringAddressWithHint(nht, "s")
		b, err := s.enc.Marshal(struct {
			A Address
		}{})
		s.NoError(err)

		return ad, b
	}
	s.decode = func(b []byte) (Address, error) {
		var u struct {
			A AddressDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &u))

		uad, err := u.A.Encode(s.enc)
		s.NoError(err)

		return uad, nil
	}
	s.compare = func(a, b Address) {
		s.NotNil(a)
		s.Nil(b)
	}

	suite.Run(t, s)
}

func TestNilStringAddressDecoderBSON(t *testing.T) {
	nht := hint.NewHint(hint.Type("uht"), "v0.0.1")

	s := new(testStringAddressEncode)
	s.enc = bsonenc.NewEncoder()
	s.encode = func() (Address, []byte) {
		ad := NewStringAddressWithHint(nht, "s")
		b, err := s.enc.Marshal(struct {
			A Address
		}{})
		s.NoError(err)

		return ad, b
	}
	s.decode = func(b []byte) (Address, error) {
		var u struct {
			A AddressDecoder
		}
		s.NoError(s.enc.Unmarshal(b, &u))

		uad, err := u.A.Encode(s.enc)
		s.NoError(err)

		return uad, nil
	}
	s.compare = func(a, b Address) {
		s.NotNil(a)
		s.Nil(b)
	}

	suite.Run(t, s)
}
