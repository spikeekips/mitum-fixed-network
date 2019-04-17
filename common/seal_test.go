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
	A       uint64
	B       string
	encoded []byte
}

func (s testSerializableForSeal) String() string {
	return fmt.Sprintf("A='%v' B='%v'", s.A, s.B)
}

func (s testSerializableForSeal) Encode() ([]byte, error) {
	var err error
	s.encoded, err = Encode(s)

	return s.encoded, err
}

func (s *testSerializableForSeal) Decode(b []byte) error {
	return Decode(b, s)
}

func (s testSerializableForSeal) Hash() (Hash, error) {
	if s.encoded == nil {
		if e, err := s.Encode(); err != nil {
			return Hash{}, err
		} else {
			s.encoded = e
		}
	}

	return NewHash("tt", s.encoded), nil
}

func (t *testSeal) TesttestSerializableForSeal() {
	s := testSerializableForSeal{A: 1, B: "b"}
	hash, err := s.Hash()
	t.NoError(err)

	encoded, err := s.Encode()
	t.NoError(err)
	t.Equal(hash.Body(), RawHash(encoded))
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
	t.Equal(bodyHash, seal.hash)

	// signature should be empty
	t.Empty(seal.Signature)

	// body
	encoded, _ := body.Encode()
	t.Equal(encoded, seal.Body)
}

func (t *testSeal) TestSign() {
	networkID := NetworkID([]byte("this-is-network"))
	body := testSerializableForSeal{A: 1, B: "b"}
	bodyHash, _ := body.Hash()

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
	err = returnedSeal.DecodeBody(&returnedBody)
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
	err = returned.DecodeBody(&sealInsideSeal)
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
		err = sealInsideSeal.DecodeBody(&sealedBody)
		t.NoError(err)

		encoded, err := body.Encode()
		t.NoError(err)

		t.Equal(encoded, sealInsideSeal.Body)

		encodedSealed, err := sealedBody.Encode()
		t.NoError(err)
		t.Equal(encoded, encodedSealed)
	}

	{
		var ts testSerializableForSeal
		err = sealInsideSeal.DecodeBody(&ts)
		t.NoError(err)

		encoded, err := ts.Encode()
		t.NoError(err)
		t.Equal(encoded, sealInsideSeal.Body)
	}
}

func TestSeal(t *testing.T) {
	suite.Run(t, new(testSeal))
}
