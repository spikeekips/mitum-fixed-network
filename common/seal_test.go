package common

import (
	"encoding/json"
	"fmt"
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

func (s testSerializableForSeal) String() string {
	return fmt.Sprintf("A='%v' B='%v'", s.A, s.B)
}

func (s testSerializableForSeal) MarshalBinary() ([]byte, error) {
	return Encode(s)
}

func (s *testSerializableForSeal) UnmarshalBinary(b []byte) error {
	return Decode(b, s)
}

func (s testSerializableForSeal) Hash() (Hash, []byte, error) {
	encoded, err := s.MarshalBinary()
	if err != nil {
		return Hash{}, nil, err
	}

	return NewHash("tt", encoded), encoded, nil
}

func (t *testSeal) TesttestSerializableForSeal() {
	s := testSerializableForSeal{A: 1, B: "b"}
	hash, encoded, err := s.Hash()
	t.NoError(err)

	t.Equal(hash.Body(), RawHash(encoded))
}

func (t *testSeal) TestNew() {
	body := testSerializableForSeal{A: 1, B: "b"}
	bodyHash, _, err := body.Hash()
	t.NoError(err)

	seal, err := NewSeal(BallotSeal, body)
	t.NoError(err)

	// check version
	t.Equal(CurrentSealVersion, seal.Version)

	// check hash
	t.Equal(bodyHash, seal.hash)

	// signature should be empty
	t.Empty(seal.Signature)

	// body
	encoded, _ := body.MarshalBinary()
	t.Equal(encoded, seal.Body)
}

func (t *testSeal) TestSign() {
	networkID := NetworkID([]byte("this-is-network"))
	body := testSerializableForSeal{A: 1, B: "b"}
	bodyHash, _, _ := body.Hash()

	seal, _ := NewSeal(BallotSeal, body)

	seed := RandomSeed()
	err := seal.Sign(networkID, seed)
	t.NoError(err)

	t.Equal(seed.Address(), seal.Source)

	// signature should not be empty
	t.NotEmpty(seal.Signature)

	expected, _ := NewSignature(networkID, seed, bodyHash)

	t.Equal(expected, seal.Signature)

	// check signature
	err = seal.CheckSignature(networkID)
	t.NoError(err)
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

	var returnedSeal Seal
	err = json.Unmarshal(b, &returnedSeal)
	t.NoError(err)

	var returnedBody testSerializableForSeal
	err = returnedSeal.UnmarshalBody(&returnedBody)
	t.NoError(err)

	t.IsType(testSerializableForSeal{}, returnedBody)
	t.NotEmpty(returnedSeal)
}

func (t *testSeal) TestJSONEmptyHash() {
	networkID := NetworkID([]byte("this-is-network"))
	body := testSerializableForSeal{A: 1, B: "b"}
	seal, _ := NewSeal(BallotSeal, body)

	seed := RandomSeed()
	err := seal.Sign(networkID, seed)
	t.NoError(err)

	seal.hash = Hash{} // make Hash to be empty

	_, err = json.MarshalIndent(seal, "", "  ")
	t.Contains(err.Error(), EmptyHashError.Message())
}

func (t *testSeal) TestSealedSeal() {
	networkID := NetworkID([]byte("this-is-network"))
	body := testSerializableForSeal{A: 1, B: "b"}

	// make new seal
	seal, _ := NewSeal(BallotSeal, body)

	seed := RandomSeed()
	err := seal.Sign(networkID, seed)
	t.NoError(err)

	_, err = json.Marshal(seal)
	t.NoError(err)

	// make new SealedSeal from seal
	sealed, err := NewSeal(SealedSeal, seal)
	t.NoError(err)

	sealedSeed := RandomSeed()
	err = sealed.Sign(networkID, sealedSeed)
	t.NoError(err)

	// check unmarshaled body is same with seal
	b, err := json.Marshal(sealed)
	t.NoError(err)

	var returned Seal
	err = json.Unmarshal(b, &returned)
	t.NoError(err)

	err = returned.CheckSignature(networkID)
	t.NoError(err)

	var sealInsideSeal Seal
	err = returned.UnmarshalBody(&sealInsideSeal)
	t.NoError(err)

	{
		t.Equal(seal.Version, sealInsideSeal.Version)
		t.Equal(seal.Type, sealInsideSeal.Type)
		t.Equal(seal.Source, sealInsideSeal.Source)
		t.Equal(seal.Signature, sealInsideSeal.Signature)
		t.Equal(seal.hash, sealInsideSeal.hash)
		t.Equal(seal.Body, sealInsideSeal.Body)
	}

	{
		var sealedBody testSerializableForSeal
		err = sealInsideSeal.UnmarshalBody(&sealedBody)
		t.NoError(err)

		encoded, err := body.MarshalBinary()
		t.NoError(err)

		t.Equal(encoded, sealInsideSeal.Body)

		encodedSealed, err := sealedBody.MarshalBinary()
		t.NoError(err)
		t.Equal(encoded, encodedSealed)
	}

	{
		var ts testSerializableForSeal
		err = sealInsideSeal.UnmarshalBody(&ts)
		t.NoError(err)

		encoded, err := ts.MarshalBinary()
		t.NoError(err)
		t.Equal(encoded, sealInsideSeal.Body)
	}
}

func TestSeal(t *testing.T) {
	suite.Run(t, new(testSeal))
}
