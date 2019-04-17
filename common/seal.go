package common

import (
	"encoding/json"
	"reflect"

	"github.com/Masterminds/semver"
)

var (
	CurrentSealVersion semver.Version = *semver.MustParse("v0.1-proto")
)

type SealType uint

const (
	_ SealType = iota
	BallotSeal
	TransactionSeal
)

func (s SealType) String() string {
	switch s {
	case BallotSeal:
		return "ballot"
	case TransactionSeal:
		return "transaction"
	default:
		return ""
	}
}

func (s SealType) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *SealType) UnmarshalJSON(b []byte) error {
	var i string
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	switch i {
	case "ballot":
		*s = BallotSeal
	case "transaction":
		*s = TransactionSeal
	default:
		return UnknownSealTypeError
	}

	return nil
}

type Seal struct {
	Type      SealType
	Version   semver.Version
	Signature Signature
	Hash      Hash
	Body      interface{}
	rawBody   json.RawMessage
}

func NewSeal(t SealType, body Hashable) (Seal, error) {
	hash, err := body.Hash()
	if err != nil {
		return Seal{}, err
	}

	return Seal{
		Type:    t,
		Version: CurrentSealVersion,
		Hash:    hash,
		Body:    body,
	}, nil
}

func (s *Seal) Sign(networkID NetworkID, seed Seed) error {
	signature, err := NewSignature(networkID, seed, s.Hash)
	if err != nil {
		return err
	}

	s.Signature = signature
	return nil
}

func (s Seal) Verify(networkID NetworkID, address Address) error {
	hash, err := s.Body.(Hashable).Hash()
	if err != nil {
		return err
	}

	return address.Verify(
		append(networkID, hash.Bytes()...),
		[]byte(s.Signature),
	)
}

func (s Seal) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"version":   &s.Version,
		"type":      s.Type,
		"signature": s.Signature,
		"hash":      s.Hash,
		"body":      s.Body,
	})
}

func (s *Seal) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	var version semver.Version
	if err := json.Unmarshal(raw["version"], &version); err != nil {
		return err
	}

	var sealType SealType
	if err := json.Unmarshal(raw["type"], &sealType); err != nil {
		return err
	}

	var signature Signature
	if err := json.Unmarshal(raw["signature"], &signature); err != nil {
		return err
	}

	var hash Hash
	if err := json.Unmarshal(raw["hash"], &hash); err != nil {
		return err
	}

	s.Version = version
	s.Type = sealType
	s.Signature = signature
	s.Hash = hash
	s.rawBody = raw["body"]

	return nil
}

func UnmarshalSeal(b []byte, i interface{}) (Seal, error) {
	var seal Seal
	if err := json.Unmarshal(b, &seal); err != nil {
		return Seal{}, err
	}

	err := json.Unmarshal(seal.rawBody, i)
	if err != nil {
		return Seal{}, err
	}

	seal.Body = reflect.ValueOf(i).Elem().Interface()

	return seal, nil
}

func (s Seal) String() string {
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b)
}
