package valuehash

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/hint"
)

type testKeccak struct {
	suite.Suite
}

func (t *testKeccak) SetupTest() {
	_ = hint.RegisterType(SHA512{}.Hint().Type(), "keccak-512-v0.1")
}

func (t *testKeccak) TestNew() {
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

func (t *testKeccak) TestJSONMarshal() {
	b := []byte("killme")
	s512 := NewSHA512(b)

	{
		b, err := MarshalJSON(s512)
		t.NoError(err)

		var jh JSONHash
		t.NoError(err, json.Unmarshal(b, &jh))

		t.Equal(s512.Hint(), jh.JSONPackHintedHead.H)
		t.Equal(s512.Bytes(), jh.Bytes())
	}

	{
		b, err := json.Marshal(s512)
		t.NoError(err)

		var jh JSONHash
		t.NoError(err, json.Unmarshal(b, &jh))

		t.Equal(s512.Hint(), jh.JSONPackHintedHead.H)
		t.Equal(s512.Bytes(), jh.Bytes())
	}
}

func TestKeccak(t *testing.T) {
	suite.Run(t, new(testKeccak))
}
