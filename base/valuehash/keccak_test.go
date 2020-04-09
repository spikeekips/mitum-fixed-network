package valuehash

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testKeccak512 struct {
	suite.Suite
}

func (t *testKeccak512) TestEmpty() {
	s := SHA512{}
	err := s.IsValid(nil)
	t.Contains(err.Error(), "empty")
}

func (t *testKeccak512) TestNil() {
	s := NewSHA512(nil)
	err := s.IsValid(nil)
	t.Contains(err.Error(), "empty")
}

func (t *testKeccak512) TestNew() {
	s512 := NewSHA512(nil)
	t.Implements((*Hash)(nil), s512)
	t.Equal(sha512Size, s512.Size())

	initial := s512.Bytes()

	b := []byte("showme")
	s512 = NewSHA512(b)

	t.NotEqual(initial, s512.Bytes())

	newS512 := NewSHA512(b)

	t.Equal(s512.Bytes(), newS512.Bytes())
}

func (t *testKeccak512) TestLoadFromBytes() {
	h := NewSHA512([]byte("showme"))

	b := h.Bytes()

	uh, err := LoadSHA512FromBytes(b)
	t.NoError(err)
	t.True(h.Equal(uh))
}

func (t *testKeccak512) TestLoadFromString() {
	b := []byte("showme")
	h := NewSHA512(b)

	s := h.String()

	uh, err := LoadSHA512FromString(s)
	t.NoError(err)
	t.True(h.Equal(uh))
}

func (t *testKeccak512) TestJSONMarshal() {
	b := []byte("killme")
	s512 := NewSHA512(b)

	{
		b, err := marshalJSON(s512)
		t.NoError(err)

		var jh JSONHash
		t.NoError(err, json.Unmarshal(b, &jh))

		t.Equal(s512.Hint(), jh.JSONPackHintedHead.H)
		t.Equal(s512.String(), jh.Hash)
	}

	{
		b, err := json.Marshal(s512)
		t.NoError(err)

		var jh JSONHash
		t.NoError(err, json.Unmarshal(b, &jh))

		t.Equal(s512.Hint(), jh.JSONPackHintedHead.H)
		t.Equal(s512.String(), jh.Hash)
	}
}

func TestKeccak512(t *testing.T) {
	suite.Run(t, new(testKeccak512))
}

type testKeccak256 struct {
	suite.Suite
}

func (t *testKeccak256) TestEmpty() {
	s := SHA256{}
	err := s.IsValid(nil)
	t.Contains(err.Error(), "empty")
}

func (t *testKeccak256) TestNil() {
	s := NewSHA256(nil)
	err := s.IsValid(nil)
	t.Contains(err.Error(), "empty")
}

func (t *testKeccak256) TestNew() {
	s256 := NewSHA256(nil)
	t.Implements((*Hash)(nil), s256)
	t.Equal(sha256Size, s256.Size())

	initial := s256.Bytes()

	b := []byte("showme")
	s256 = NewSHA256(b)

	t.NotEqual(initial, s256.Bytes())

	newS256 := NewSHA256(b)

	t.Equal(s256.Bytes(), newS256.Bytes())
}

func (t *testKeccak256) TestJSONMarshal() {
	b := []byte("killme")
	s256 := NewSHA256(b)

	{
		b, err := marshalJSON(s256)
		t.NoError(err)

		var jh JSONHash
		t.NoError(err, json.Unmarshal(b, &jh))

		t.Equal(s256.Hint(), jh.JSONPackHintedHead.H)
		t.Equal(s256.String(), jh.Hash)
	}

	{
		b, err := json.Marshal(s256)
		t.NoError(err)

		var jh JSONHash
		t.NoError(err, json.Unmarshal(b, &jh))

		t.Equal(s256.Hint(), jh.JSONPackHintedHead.H)
		t.Equal(s256.String(), jh.Hash)
	}
}

func (t *testKeccak256) TestLoadFromBytes() {
	h := NewSHA256([]byte("showme"))

	b := h.Bytes()

	uh, err := LoadSHA256FromBytes(b)
	t.NoError(err)
	t.True(h.Equal(uh))
}

func (t *testKeccak256) TestLoadFromString() {
	b := []byte("showme")
	h := NewSHA256(b)

	s := h.String()

	uh, err := LoadSHA256FromString(s)
	t.NoError(err)
	t.True(h.Equal(uh))
}

func TestKeccak256(t *testing.T) {
	suite.Run(t, new(testKeccak256))
}
