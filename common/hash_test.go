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

	hash := NewHash(hint, body)

	t.Equal(hint, hash.Hint())

	raw := RawHash(body)
	t.Equal(raw, hash.Body())
	t.Equal(raw[:], hash.Bytes())
}

func (t *testHash) TestString() {
	hint := "tx"
	body := []byte("findme")

	hash := NewHash(hint, body)

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

func TestHash(t *testing.T) {
	suite.Run(t, new(testHash))
}
