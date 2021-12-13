package valuehash

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testBlake3256 struct {
	suite.Suite
}

func (t *testBlake3256) TestNew() {
	h := NewBlake3256(nil)
	t.Implements((*Hash)(nil), h)

	initial := h.Bytes()

	b := []byte("showme")
	h = NewBlake3256(b)

	t.T().Log(h.String())
	t.Equal("5DeBwvk8DuUriS3ppjvh31AAYfB7UcmfQjuFV2z9kroj", h.String())

	t.NotEqual(initial, h.Bytes())

	newS512 := NewBlake3256(b)

	t.Equal(h.Bytes(), newS512.Bytes())
}

func TestBlake3256(t *testing.T) {
	suite.Run(t, new(testBlake3256))
}
