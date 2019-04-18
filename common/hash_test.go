package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testHash struct {
	suite.Suite
}

func (t *testHash) TestNew() {
	hint := "tx"
	body := []byte("findme")

	hash, err := NewHash(hint, body)
	t.NoError(err)

	t.Equal(hint, hash.Hint())

	raw := RawHash(body)
	t.Equal(raw, hash.Body())
	t.Equal(raw[:], hash.Bytes())
}

func (t *testHash) TestString() {
	hint := "tx"
	body := []byte("findme")

	hash, err := NewHash(hint, body)
	t.NoError(err)

	t.Equal("tx-JAdmEqVfoBGitPN386jVGhKGtF6tAhQZVSaAze8DPD1M", hash.String())
}

func (t *testHash) TestJSON() {
	hint := "tx"
	body := []byte("findme")

	hash, err := NewHashFromObject(hint, body)
	t.NoError(err)

	b, err := json.Marshal(hash)
	t.NoError(err)

	var returned Hash
	err = json.Unmarshal(b, &returned)
	t.NoError(err)

	t.Equal(hash, returned)
	t.True(hash.Equal(returned))
}

func (t *testHash) TestJSONEmptyHash() {
	hash := Hash{}
	_, err := json.Marshal(hash)
	t.Contains(err.Error(), EmptyHashError.Message())
}

func (t *testHash) TestBinaryMarshal() {
	hint := "sl"
	body := []byte("showme")
	hash, _ := NewHashFromObject(hint, body)

	b, err := hash.MarshalBinary()
	t.NoError(err)

	var newHash Hash
	err = newHash.UnmarshalBinary(b)
	t.NoError(err)

	t.Equal(hash.h, newHash.h)
	t.Equal(hash.b, newHash.b)
	t.True(hash.Equal(newHash))
}

func TestHash(t *testing.T) {
	suite.Run(t, new(testHash))
}
