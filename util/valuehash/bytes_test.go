package valuehash

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testBytes struct {
	suite.Suite
}

func (t *testBytes) TestEmpty() {
	s := Bytes{}
	err := s.IsValid(nil)
	t.Contains(err.Error(), "empty")
}

func (t *testBytes) TestNil() {
	s := NewBytes(nil)
	err := s.IsValid(nil)
	t.Contains(err.Error(), "empty")
}

func (t *testBytes) TestEqual() {
	hs := RandomSHA256()
	bhs := NewBytes(hs.Bytes())

	t.True(hs.Equal(bhs))
}

func (t *testBytes) TestNew() {
	hs := NewBytes(nil)
	t.Implements((*Hash)(nil), hs)
	t.Equal(0, hs.Size())

	initial := hs.Bytes()

	b := []byte("showme")
	hs = NewBytes(b)

	t.NotEqual(initial, hs.Bytes())

	newdm := NewBytes(b)

	t.Equal(hs.Bytes(), newdm.Bytes())
	t.Equal(len(b), newdm.Size())
	t.Equal(b, newdm.Bytes())
}

func (t *testBytes) TestJSONMarshal() {
	b := []byte("killme")
	hs := NewBytes(b)

	{
		b, err := json.Marshal(hs)
		t.NoError(err)

		var jh Bytes
		t.NoError(err, json.Unmarshal(b, &jh))

		t.Equal(hs.Hint(), jh.Hint())
		t.Equal(hs.String(), jh.String())
		t.True(hs.Equal(jh))
	}
}

func TestBytes(t *testing.T) {
	suite.Run(t, new(testBytes))
}
