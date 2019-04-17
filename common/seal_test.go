package common

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type testSeal struct {
	suite.Suite
}

type testSerializableForSeal struct {
	A uint64
	B string
}

func (s testSerializableForSeal) Hash() (Hash, error) {
	return NewHashFromObject("tt", s)
}

func (t *testSeal) TestNew() {
	body := testSerializableForSeal{A: 1, B: "b"}
	bodyHash, err := body.Hash()
	t.NoError(err)

	seal, err := NewSeal(BallotSeal, body)
	t.NoError(err)

	// check version
	t.Equal(CurrentSealVersion, seal.Version)

	// check hash
	t.Equal(bodyHash, seal.Hash)

	// signature should be empty
	t.Empty(seal.Signature)

	// body
	t.Equal(body, seal.Body)
}

func (t *testSeal) TestSign() {
	networkID := NetworkID([]byte("this-is-network"))
	body := testSerializableForSeal{A: 1, B: "b"}
	bodyHash, _ := body.Hash()

	seal, _ := NewSeal(BallotSeal, body)

	seed := RandomSeed()
	err := seal.Sign(networkID, seed)
	t.NoError(err)

	// signature should not be empty
	t.NotEmpty(seal.Signature)

	expected, _ := NewSignature(networkID, seed, bodyHash)

	t.Equal(expected, seal.Signature)
}

func (t *testSeal) TestJSON() {
	networkID := NetworkID([]byte("this-is-network"))
	body := testSerializableForSeal{A: 1, B: "b"}
	seal, _ := NewSeal(BallotSeal, body)

	seed := RandomSeed()
	err := seal.Sign(networkID, seed)
	t.NoError(err)

	b, err := json.MarshalIndent(seal, "", "  ")
	t.NoError(err)

	var returnedBody testSerializableForSeal
	returnedSeal, err := UnmarshalSeal(b, &returnedBody)
	t.NoError(err)
	t.IsType(testSerializableForSeal{}, returnedBody)
	t.NotEmpty(returnedSeal)
}

func TestSeal(t *testing.T) {
	suite.Run(t, new(testSeal))
}
