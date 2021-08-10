package valuehash

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testBlake3256 struct {
	suite.Suite
}

func (t *testBlake3256) TestEmpty() {
	s := Blake3256{}
	err := s.IsValid(nil)
	t.Contains(err.Error(), "empty")
}

func (t *testBlake3256) TestNil() {
	s := NewBlake3256(nil)
	err := s.IsValid(nil)
	t.Contains(err.Error(), "empty")
}

func (t *testBlake3256) TestNew() {
	hs := NewBlake3256(nil)
	t.Implements((*Hash)(nil), hs)

	initial := hs.Bytes()

	b := []byte("showme")
	hs = NewBlake3256(b)

	t.T().Log(hs.String())
	t.Equal("5DeBwvk8DuUriS3ppjvh31AAYfB7UcmfQjuFV2z9kroj", hs.String())

	t.NotEqual(initial, hs.Bytes())

	newS512 := NewBlake3256(b)

	t.Equal(hs.Bytes(), newS512.Bytes())
}

func (t *testBlake3256) TestJSONMarshal() {
	b := []byte("killme")
	hs := NewBlake3256(b)

	{
		b, err := marshalJSON(hs)
		t.NoError(err)

		var jh Bytes
		t.NoError(err, json.Unmarshal(b, &jh))

		t.True(hs.Equal(jh))
	}
}

func TestBlake3256(t *testing.T) {
	suite.Run(t, new(testBlake3256))
}
