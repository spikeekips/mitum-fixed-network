package valuehash

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spikeekips/mitum/util"
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

func (t *testBytes) TestSHA256WithPrefix() {
	h := RandomSHA256WithPrefix([]byte("showme"))
	t.NoError(h.IsValid(nil))

	h = RandomSHA256WithPrefix(bytes.Repeat([]byte("s"), 53)) // NOTE 52 is max prefix length
	err := h.IsValid(nil)
	t.Contains(err.Error(), "invalid hash")
	t.Contains(err.Error(), "over max")

	{
		prefix := []byte("findme")
		h = RandomSHA256WithPrefix(prefix)
		t.NoError(h.IsValid(nil))

		// NOTE decode prefix random hash
		bh := h.Bytes()
		lh, err := util.BytesToInt64(bh[:8])
		t.NoError(err)
		lp, err := util.BytesToInt64(bh[8:16])
		t.NoError(err)

		t.Equal(int64(sha256Size), lh)
		t.Equal(int64(len(prefix)), lp)

		pb := bh[16+lh:]

		t.Equal(prefix, pb)
	}
}

func (t *testBytes) TestSHA512WithPrefix() {
	h := RandomSHA512WithPrefix([]byte("showme"))
	t.NoError(h.IsValid(nil))

	h = RandomSHA512WithPrefix(bytes.Repeat([]byte("s"), 21)) // NOTE 20 is max prefix length
	err := h.IsValid(nil)
	t.Contains(err.Error(), "invalid hash")
	t.Contains(err.Error(), "over max")

	{
		prefix := []byte("findme")
		h = RandomSHA512WithPrefix(prefix)
		t.NoError(h.IsValid(nil))

		// NOTE decode prefix random hash
		bh := h.Bytes()
		lh, err := util.BytesToInt64(bh[:8])
		t.NoError(err)
		lp, err := util.BytesToInt64(bh[8:16])
		t.NoError(err)

		t.Equal(int64(sha512Size), lh)
		t.Equal(int64(len(prefix)), lp)

		pb := bh[16+lh:]

		t.Equal(prefix, pb)
	}
}

func (t *testBytes) TestJSONMarshal() {
	b := []byte("killme")
	hs := NewBytes(b)

	{
		b, err := json.Marshal(hs)
		t.NoError(err)

		var jh Bytes
		t.NoError(err, json.Unmarshal(b, &jh))

		t.Equal(hs.String(), jh.String())
		t.True(hs.Equal(jh))
	}
}

func TestBytes(t *testing.T) {
	suite.Run(t, new(testBytes))
}
