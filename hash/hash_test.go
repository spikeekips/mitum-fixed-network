package hash

import (
	"testing"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testHash struct {
	suite.Suite
}

func (t *testHash) TestNew() {
	hint := "block"
	value := []byte("show me")

	hash, err := NewHash(hint, value)
	t.NoError(err)
	t.NotEmpty(hash)

	_, ok := interface{}(hash).(Hash)
	t.True(ok)

	t.Equal(`block:5NfGRdg6ex`, hash.String())
}

func (t *testHash) TestEqual() {
	hint := "block"

	hash0, err := NewHash(hint, []byte("show me"))
	t.NoError(err)
	hash1, err := NewHash(hint, []byte("show me"))
	t.NoError(err)
	hash2, err := NewHash(hint, []byte("findme me"))
	t.NoError(err)

	t.True(hash0.Equal(hash1))
	t.True(hash1.Equal(hash0))
	t.False(hash0.Equal(hash2))
}

func (t *testHash) TestIsValid() {
	{
		_, err := NewHash("", []byte("show me")) // hint should be not empty
		t.Contains(err.Error(), "zero hint length")
	}

	{
		hash, err := NewHash("hint", []byte("show me"))
		t.NoError(err)
		t.NoError(hash.IsValid())
	}
}

func (t *testHash) TestMarshal() {
	hash, err := NewHash("hint", []byte("show me"))
	t.NoError(err)

	b, err := rlp.EncodeToBytes(hash)
	t.NoError(err)

	var uhash Hash
	err = rlp.DecodeBytes(b, &uhash)
	t.NoError(err)

	t.NoError(uhash.IsValid())
	t.True(hash.Equal(uhash))
}

func (t *testHash) TestUnmarshal() {
	b := []byte("findme")

	var uhash Hash
	err := rlp.DecodeBytes(b, &uhash)
	t.True(xerrors.Is(InvalidHashInputError, err))
}

func (t *testHash) TestNilHash() {
	b := base58.Decode("N1LHASH")
	t.Equal(nilBody[:], b)

	h := NilHash("block")
	t.NoError(h.IsValid())
	t.True(h.IsNil())
	t.Equal("block:N1LHASH", h.String())
}

func (t *testHash) TestFromString() {
	{
		s := "bk:6kpjZZfXn9b8m2qx37xS5pvBH16BLEvYwoZA2WvfQw4s"
		h, err := NewHashFromString(s)
		t.NoError(err)
		t.Equal("bk", h.Hint())
		t.Equal(s, h.String())
	}
}

func TestHash(t *testing.T) {
	suite.Run(t, new(testHash))
}
