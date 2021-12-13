package valuehash

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testKeccak512 struct {
	suite.Suite
}

func (t *testKeccak512) TestNew() {
	h := NewSHA512(nil)
	t.Implements((*Hash)(nil), h)

	initial := h.Bytes()

	b := []byte("showme")
	h = NewSHA512(b)

	t.T().Log(h.String())
	t.Equal("2H76oz198raDfKuuZR2UzykD5ahnWkki5QHN7qaThedyt9KqL5bg3CkW3r49Ahto8LikRhP9dC4QvG6t5C2WoFoc", h.String())

	t.NotEqual(initial, h.Bytes())

	newS512 := NewSHA512(b)

	t.Equal(h.Bytes(), newS512.Bytes())
}

func TestKeccak512(t *testing.T) {
	suite.Run(t, new(testKeccak512))
}

type testKeccak256 struct {
	suite.Suite
}

func (t *testKeccak256) TestNew() {
	h := NewSHA256(nil)
	t.Implements((*Hash)(nil), h)

	initial := h.Bytes()

	b := []byte("showme")
	h = NewSHA256(b)

	t.T().Log(h.String())
	t.Equal("67fQPFDYM4QJjdCuJhM4EUakDPK6uRa4TfF1qzMNi5XV", h.String())

	t.NotEqual(initial, h.Bytes())

	newS256 := NewSHA256(b)

	t.Equal(h.Bytes(), newS256.Bytes())
}

func TestKeccak256(t *testing.T) {
	suite.Run(t, new(testKeccak256))
}
