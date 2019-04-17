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
	prefix := "tx"
	body := []byte("findme")

	hash := NewHash(prefix, body)

	t.Equal(prefix, hash.Prefix())

	raw := RawHash(body)
	t.Equal(raw, hash.Body())
	t.Equal(raw[:], hash.Bytes())
}

func (t *testHash) TestString() {
	prefix := "tx"
	body := []byte("findme")

	hash := NewHash(prefix, body)

	t.Equal("tx-JAdmEqVfoBGitPN386jVGhKGtF6tAhQZVSaAze8DPD1M", hash.String())
}

func (t *testHash) TestJSON() {
	prefix := "tx"
	body := []byte("findme")

	hash, err := NewHashFromObject(prefix, body)
	t.NoError(err)

	b, err := json.Marshal(hash)
	t.NoError(err)

	var returned Hash
	err = json.Unmarshal(b, &returned)
	t.NoError(err)

	t.Equal(hash, returned)
	t.True(hash.Equal(returned))
}

func TestHash(t *testing.T) {
	suite.Run(t, new(testHash))
}
