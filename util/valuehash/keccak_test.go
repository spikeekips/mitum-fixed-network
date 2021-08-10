package valuehash

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
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
	hs := NewSHA512(nil)
	t.Implements((*Hash)(nil), hs)

	initial := hs.Bytes()

	b := []byte("showme")
	hs = NewSHA512(b)

	t.NotEqual(initial, hs.Bytes())

	newS512 := NewSHA512(b)

	t.Equal(hs.Bytes(), newS512.Bytes())
}

func (t *testKeccak512) TestJSONMarshal() {
	b := []byte("killme")
	hs := NewSHA512(b)

	{
		b, err := marshalJSON(hs)
		t.NoError(err)

		var jh Bytes
		t.NoError(err, json.Unmarshal(b, &jh))

		t.True(hs.Equal(jh))
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
	hs := NewSHA256(nil)
	t.Implements((*Hash)(nil), hs)

	initial := hs.Bytes()

	b := []byte("showme")
	hs = NewSHA256(b)

	t.NotEqual(initial, hs.Bytes())

	newS256 := NewSHA256(b)

	t.Equal(hs.Bytes(), newS256.Bytes())
}

func (t *testKeccak256) TestBSONMarshal() {
	hs := NewSHA256([]byte("killme"))

	_, b, err := bson.MarshalValue(hs)
	t.NoError(err)

	uh, err := unmarshalBSONValue(b)
	t.NoError(err)

	t.True(hs.Equal(uh))
}

func (t *testKeccak256) TestJSONMarshal() {
	b := []byte("killme")
	hs := NewSHA256(b)

	{
		b, err := marshalJSON(hs)
		t.NoError(err)

		var jh Bytes
		t.NoError(err, json.Unmarshal(b, &jh))

		t.True(hs.Equal(jh))
	}
}

func TestKeccak256(t *testing.T) {
	suite.Run(t, new(testKeccak256))
}
